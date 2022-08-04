/// Allows running an OONIEngine in a background isolate and
/// interacting with it by exchanging messages.
import "dart:async";
import "dart:ffi";
import "dart:io";
import "dart:isolate";

import "package:async/async.dart";
import "package:ffi/ffi.dart";
import 'package:ooniengine/bindings.dart';
import 'package:ooniengine/abi.dart';

/// An event emitted by the OONI Engine API.
class Event {
  /// The name of the event.
  String name = "";

  /// The event value as a serialized JSON struct.
  String value = "";
}

/// OONIEngineFFI wrapper with better typing.
class _EngineWrapper {
  /// Whether to print messages exchanged with the engine.
  final bool _debug = Platform.environment["OONI_ENGINE_DEBUG"] == "1";

  /// The underlying engine.
  final OONIEngineFFI _engine;

  /// Constructs a new instance using [lib], which must be
  /// the absolute path to the OONIEngine DLL.
  _EngineWrapper(String lib)
      : _engine = OONIEngineFFI(DynamicLibrary.open(lib));

  /// OONITaskStart wrapper. Returns null in case of failure.
  int taskStart(String taskName, String taskArguments) {
    if (_debug) {
      print("ENGINE: > ${ABIVersion} ${taskName} ${taskArguments}");
    }
    final abi = ABIVersion.toNativeUtf8().cast<Char>();
    final name = taskName.toNativeUtf8().cast<Char>();
    final arguments = taskArguments.toNativeUtf8().cast<Char>();
    final taskID = _engine.OONITaskStart(abi, name, arguments);
    malloc.free(abi);
    malloc.free(name);
    malloc.free(arguments);
    if (_debug) {
      print("ENGINE: < ${taskID}");
    }
    return taskID;
  }

  /// OONITaskWaitForNextEvent wrapper. The [timeout] is the time to
  /// wait for the next event in milliseconds. A null return value
  /// indicates a timeout, that the task is done, or other unlikely
  /// internal error conditions. Use isDone to know when the task
  /// is done rather than relying on this method's retval.
  Event? taskWaitForNextEvent(int taskID, int timeout) {
    if (_debug) {
      print("ENGINE: > WAIT ${taskID} ${timeout}");
    }
    final tev = _engine.OONITaskWaitForNextEvent(taskID, timeout);
    if (tev == nullptr) {
      if (_debug) {
        print("ENGINE: < null");
      }
      return null; // timeout or other kind of errors
    }
    var event = Event();
    event.name = tev.ref.Name.cast<Utf8>().toDartString();
    event.value = tev.ref.Value.cast<Utf8>().toDartString();
    _engine.OONIEventFree(tev);
    if (_debug) {
      print("ENGINE: < ${event.name} ${event.value}");
    }
    return event;
  }

  /// OONITaskIsDone wrapper.
  bool taskIsDone(int taskID) {
    final done = _engine.OONITaskIsDone(taskID) != 0;
    if (_debug) {
      print("ENGINE: < DONE ${done}");
    }
    return done;
  }

  /// OONITaskInterrupt wrapper.
  void taskInterrupt(int taskID) {
    if (_debug) {
      print("ENGINE: > INTERRUPT ${taskID}");
    }
    _engine.OONITaskInterrupt(taskID);
  }

  /// OONITaskFree wrapper.
  void taskFree(int taskID) {
    if (_debug) {
      print("ENGINE: > FREE ${taskID}");
    }
    _engine.OONITaskFree(taskID);
  }
}

/// Engine running as a service inside a separate Isolate.
class _EngineService {
  /// Absolute path of the OONIEngine DLL.
  final String _lib;

  /// The sender connected to the controller's receiver.
  final SendPort _sender;

  /// Construct this instance with the given path to
  /// the OONI engine lib and the given SendPort.
  _EngineService(this._lib, this._sender);

  /// Runs the service until the parent Isolate terminates. The
  /// protocol is such that the first message sent over the sender
  /// contains the port to send messages to this Isolate.
  void run(_) async {
    final engine = _EngineWrapper(_lib);
    final receiver = ReceivePort();
    _sender.send(receiver.sendPort); // as documented
    await for (final msg in receiver) {
      if (msg is _TaskStartRequest) {
        final task = engine.taskStart(msg.taskName, msg.taskArguments);
        _sender.send(_TaskStartResponse(task));
        continue;
      }
      if (msg is _TaskWaitForNextEventRequest) {
        final event = engine.taskWaitForNextEvent(msg.taskID, msg.timeout);
        _sender.send(_TaskWaitForNextEventResponse(event));
        continue;
      }
      if (msg is _TaskIsDoneRequest) {
        final isDone = engine.taskIsDone(msg.taskID);
        _sender.send(_TaskIsDoneResponse(isDone));
        continue;
      }
      if (msg is _TaskInterruptRequest) {
        engine.taskInterrupt(msg.taskID);
        continue;
      }
      if (msg is _TaskFreeRequest) {
        engine.taskFree(msg.taskID);
        continue;
      }
      if (msg is _ShutdownServiceRequest) {
        _sender.send(_ShutdownServiceResponse());
        return;
      }
    }
  }
}

/// Request to start a new task.
class _TaskStartRequest {
  /// Contains the task name;
  final String taskName;

  /// Contains the task arguments;
  final String taskArguments;

  /// Creates a new instance.
  _TaskStartRequest(this.taskName, this.taskArguments);
}

/// Result of starting a new task.
class _TaskStartResponse {
  /// Unique identifier of the task.
  final int taskID;

  /// Creates a new instance.
  _TaskStartResponse(this.taskID);
}

/// Request to wait for the next task event.
class _TaskWaitForNextEventRequest {
  /// Unique identifier of the task.
  final int taskID;

  /// Wait timeout in milliseconds.
  final int timeout;

  /// Creates a new instance.
  _TaskWaitForNextEventRequest(this.taskID, this.timeout);
}

/// Result of waiting for the next task event.
class _TaskWaitForNextEventResponse {
  /// Optional result (a null event implies timeout or other errors).
  final Event? event;

  /// Creates a new instance.
  _TaskWaitForNextEventResponse(this.event);
}

/// Request to check whether a task is done.
class _TaskIsDoneRequest {
  /// Unique identifier of the task.
  final int taskID;

  /// Creates a new instance.
  _TaskIsDoneRequest(this.taskID);
}

/// Result of checking whether a task is done.
class _TaskIsDoneResponse {
  /// Whether the task is done.
  final bool isDone;

  /// Creates a new instance.
  _TaskIsDoneResponse(this.isDone);
}

/// Request to interrupt a given task.
class _TaskInterruptRequest {
  /// Unique identifier of the task.
  final int taskID;

  /// Creates a new instance.
  _TaskInterruptRequest(this.taskID);
}

/// Request to free all memory associated with a task.
class _TaskFreeRequest {
  /// Unique identifier of the task.
  final int taskID;

  /// Creates a new instance.
  _TaskFreeRequest(this.taskID);
}

/// Request to shutdown the background service.
class _ShutdownServiceRequest {}

/// Response to a request to shutdown the background service.
class _ShutdownServiceResponse {}

/// Allows running an OONIEngine inside a background Isolate and
/// interacting with it by proxying OONIEngine API methods.
///
/// Each invoked method will send a message to the background
/// Isolate and receive the corresponding response.
///
/// To avoid blocking the background isolate inside a C call
/// for long period of times, it's recommended to issue
/// taskWaitForNextEvent calls with a small, positive timeout.
class Engine {
  /// Whether we have been initialized.
  bool _initialized = false;

  /// Absolute path to the OONIEngine DLL.
  final String _lib;

  /// Internal reference to the sender. Do not use directly,
  /// rather use _getSender() to obtain a sender.
  late SendPort _internalSender;

  /// Receives messages from the background isolate.
  late StreamQueue<dynamic> _receiver;

  /// Constructs using the given library path, which must be the
  /// absolute path to the OONIEngine DLL.
  Engine(this._lib);

  /// Ensures we have started the background isolate
  /// and returns the sender to speak with it.
  Future<SendPort> _getSender() async {
    if (!_initialized) {
      final receiver = ReceivePort();
      _receiver = StreamQueue<dynamic>(receiver);
      final service = _EngineService(_lib, receiver.sendPort);
      await Isolate.spawn(service.run, Null);
      // the protocol is that the isolate sends us the sender for its
      // receiver as the first message directed to us.
      _internalSender = await _receiver.next as SendPort;
      _initialized = true;
    }
    return _internalSender;
  }

  /// Invokes OONITaskStart with the given [taskName] and [taskArguments].
  Future<int> taskStart(String taskName, String taskArguments) async {
    final sender = await _getSender();
    sender.send(_TaskStartRequest(taskName, taskArguments));
    final r = await _receiver.next as _TaskStartResponse;
    return r.taskID;
  }

  /// Invokes OONITaskWaitForNextEvent using the given [taskID] and
  /// [timeout], which is expressed in milliseconds.
  Future<Event?> taskWaitForNextEvent(int taskID, int timeout) async {
    final sender = await _getSender();
    sender.send(_TaskWaitForNextEventRequest(taskID, timeout));
    final resp = await _receiver.next as _TaskWaitForNextEventResponse;
    return resp.event;
  }

  /// Invokes OONITaskIsDone.
  Future<bool> taskIsDone(int taskID) async {
    final sender = await _getSender();
    sender.send(_TaskIsDoneRequest(taskID));
    final resp = await _receiver.next as _TaskIsDoneResponse;
    return resp.isDone;
  }

  /// Invokes OONITaskInterrupt.
  void taskInterrupt(int taskID) async {
    final sender = await _getSender();
    sender.send(_TaskInterruptRequest(taskID));
  }

  /// Invokes OONITaskFree.
  void taskFree(int taskID) async {
    if (!_initialized || taskID < 0) {
      return;
    }
    final sender = await _getSender();
    sender.send(_TaskFreeRequest(taskID));
  }

  /// Unregisters the reading port. You must call this function
  /// from your main or toplevel code to make dart exit. It won't
  /// exit otherwise because it keeps reading from the port.
  void shutdown() async {
    if (!_initialized) {
      return;
    }
    final sender = await _getSender();
    sender.send(_ShutdownServiceRequest());
    await _receiver.next as _ShutdownServiceResponse;
    _receiver.cancel(immediate: true);
    _initialized = false;
  }
}
