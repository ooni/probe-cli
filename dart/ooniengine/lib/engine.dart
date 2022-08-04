/// Allows running an OONIEngine in a background isolate and
/// interacting with it by exchanging messages.
import "dart:async";
import "dart:ffi";
import "dart:io";
import "dart:isolate";
import 'dart:typed_data';

import "package:async/async.dart";
import "package:ffi/ffi.dart";
import 'package:ooniengine/abi.pb.dart';
import 'package:ooniengine/bindings.dart';
import 'package:protobuf/protobuf.dart' as $pb;

/// Low-level interface to the OONI engine.
class LLEngine {
  /// Underlying engine.
  final OONIEngineFFI _engine;

  /// Constructor.
  LLEngine(this._engine);

  /// Wrapper for OONITaskStart. The [name] argument contains the
  /// task name. The [args] argument contains the corresponding
  /// protobuf message containing arguments for the task to run.
  ///
  /// A negative return value indicates OONITaskStart failed. In such
  /// case, OONITaskStart prints diagnostic messages on the stdout.
  ///
  /// We let exceptions unwind the stack. In case of exceptions,
  /// this function does not leak any heap allocated memory.
  int taskStart(String name, $pb.GeneratedMessage args) {
    Pointer<Uint8> base = nullptr;
    Pointer<Char> nname = nullptr;
    int taskID = -1;
    try {
      nname = name.toNativeUtf8().cast<Char>(); // 🤓 needs free
      final writer = $pb.CodedBufferWriter();
      args.writeToCodedBufferWriter(writer);
      final len = writer.lengthInBytes;
      base = malloc<Uint8>(len); // 🤓 needs free
      final memview = base.asTypedList(len);
      writer.writeTo(memview); // 🤓 first copy (Go will copy again)
      final vbase = base.cast<Void>();
      taskID = _engine.OONITaskStart(nname, vbase, len);
    } finally {
      malloc.free(nname);
      malloc.free(base);
    }
    return taskID;
  }

  /// OONITaskWaitForNextEvent wrapper. The [taskID] argument is
  /// the unique task identifier returned by OONITaskStart. The
  /// [timeout] argument contains the timeout in milliseconds. Use
  /// a negative [timeout] to prevent OONITaskWaitForNextEvent from
  /// timing out. A null return value indicates a timeout, that
  /// the task is done, or any unlikely OONITaskWaitForNextEvent
  /// internal error. To know whether a task is actually done you
  /// should use OONITaskIsDone. We let exceptions unwind the stack
  /// but we also ensure we don't leak any memory on exception.
  $pb.GeneratedMessage? taskWaitForNextEvent(int taskID, int timeout) {
    $pb.GeneratedMessage? rev = null;
    final ev = _engine.OONITaskWaitForNextEvent(taskID, timeout);
    try {
      if (ev != nullptr && ev.ref.Len >= 1) {
        final name = ev.ref.Name.cast<Utf8>().toDartString(); // 🤓 small
        final len = ev.ref.Len;
        final mv = ev.ref.Base.cast<Uint8>().asTypedList(len); // 🤓 zero copy
        rev = _parseMessage(name, mv);
      }
    } finally {
      _engine.OONIEventFree(ev); // 🤓 API is robust to nullptr
    }
    return rev;
  }

  /// Parses from [data] the protobuf message corresponding to [name].
  $pb.GeneratedMessage? _parseMessage(String name, Uint8List data) {
    switch (name) {
      case "GeoIP":
        return GeoIPEvent.fromBuffer(data);
      case "Log":
        return LogEvent.fromBuffer(data);
      case "ExprimentMetaInfo":
        return ExperimentMetaInfoEvent.fromBuffer(data);
      case "Progress":
        return ProgressEvent.fromBuffer(data);
      case "DataUsage":
        return DataUsageEvent.fromBuffer(data);
      case "Submit":
        return SubmitEvent.fromBuffer(data);
      case "ExperimentMetaInfo":
        return ExperimentMetaInfoEvent.fromBuffer(data);
      default:
        stderr.write("LLEngine._parseMessage: unhandled name: ${name}\n");
        return null;
    }
  }

  /// Calls OONITaskIsDone to determine whether the task with [taskID]
  /// is done. A task is done when (1) it has stopped and (2) we've read
  /// all the emitted events using OONITaskWaitForNextEvent.
  bool taskIsDone(int taskID) {
    return _engine.OONITaskIsDone(taskID) != 0;
  }

  /// Calls OONITaskInterrupt to interrupt the task with [taskID]. An
  /// interrupted task will try to stop running ASAP.
  void taskInterrupt(int taskID) {
    _engine.OONITaskInterrupt(taskID);
  }

  /// Calls OONITaskFree to forget about the task with [taskID]. The
  /// OONI engine will internally interrupt the task and drain its
  /// events queue for us. That is, a task might continue to run for
  /// some time inside the engine after you call this method.
  void taskFree(int taskID) {
    _engine.OONITaskFree(taskID);
  }
}

/// An [LLEngine] instance running as a service inside an Isolate.
class IsolatedEngine {
  /// Absolute path of the OONIEngine DLL.
  final String _lib;

  /// The sender connected to the controller's receiver.
  final SendPort _sender;

  /// Constructor.
  IsolatedEngine(this._lib, this._sender);

  /// Runs the service. The protocol is such that the first message
  /// sent over the sender contains the port to send messages back to
  /// this service. Subsequent messages will be replies to messages
  /// we receive from our controller.
  void run(_) async {
    final engine = LLEngine(OONIEngineFFI(DynamicLibrary.open(_lib)));
    final receiver = ReceivePort();
    _sender.send(receiver.sendPort); // as documented
    await for (final msg in receiver) {
      final shouldStop = _dispatch(engine, msg);
      if (shouldStop) {
        return;
      }
    }
  }

  /// Performs the proper action using [engine] depending on [msg] type
  /// and writes the corresponding response back to our controller.
  bool _dispatch(LLEngine engine, dynamic msg) {
    if (msg is _TaskStartRequest) {
      final task = engine.taskStart(msg.name, msg.args);
      _sender.send(_TaskStartResponse(task));
      return false;
    }
    if (msg is _TaskWaitForNextEventRequest) {
      final event = engine.taskWaitForNextEvent(msg.taskID, msg.timeout);
      _sender.send(_TaskWaitForNextEventResponse(event));
      return false;
    }
    if (msg is _TaskIsDoneRequest) {
      final isDone = engine.taskIsDone(msg.taskID);
      _sender.send(_TaskIsDoneResponse(isDone));
      return false;
    }
    if (msg is _TaskInterruptRequest) {
      engine.taskInterrupt(msg.taskID);
      return false;
    }
    if (msg is _TaskFreeRequest) {
      engine.taskFree(msg.taskID);
      return false;
    }
    if (msg is _ShutdownServiceRequest) {
      _sender.send(_ShutdownServiceResponse());
      return true;
    }
    return false;
  }
}

/// Request to start a new task.
class _TaskStartRequest {
  /// Task name.
  final String name;

  /// Task arguments.
  final $pb.GeneratedMessage args;

  /// Creates a new instance.
  _TaskStartRequest(this.name, this.args);
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
  final $pb.GeneratedMessage? event;

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

/// Exception thrown by the [Engine].
class EngineException implements Exception {
  /// The cause of the exception.
  String cause;

  /// Constructor.
  EngineException(this.cause);
}

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

  /// Internal reference to the sender. Do not use directly,
  /// rather use _getSender() to obtain a sender.
  late SendPort _internalSender;

  /// Absolute path to the OONIEngine DLL.
  final String _lib;

  /// Receives messages from the background isolate.
  late StreamQueue<dynamic> _receiver;

  /// Turns on debugging mode.
  bool debug = Platform.environment["OONI_ENGINE_DEBUG"] == "1";

  /// Constructs using the given library path, which must be the
  /// absolute path to the OONIEngine DLL. The [debug] argument
  /// allows to construct an engine that logs messages exchanged
  /// with the background isolate running the real engine.
  Engine(this._lib);

  /// Ensures we have started the background isolate
  /// and returns the sender to speak with it.
  Future<SendPort> _getSender() async {
    if (!_initialized) {
      final receiver = ReceivePort();
      _receiver = StreamQueue<dynamic>(receiver);
      final service = IsolatedEngine(_lib, receiver.sendPort);
      await Isolate.spawn(service.run, Null);
      // the protocol is that the isolate sends us the sender for its
      // receiver as the first message directed to us.
      _internalSender = await _receiver.next as SendPort;
      _initialized = true;
    }
    return _internalSender;
  }

  /// Starts a new GeoIP task. Throws [EngineException] on error.
  Future<Task> startGeoIPTask(GeoIPConfig args) async {
    return _taskStart("GeoIP", args);
  }

  /// Starts a new Nettest task. Throws [EngineException] on error.
  Future<Task> startNettestTask(NettestConfig args) async {
    return _taskStart("Nettest", args);
  }

  /// Starts a new OONIRunV2MeasureDescriptor task.
  /// Throws [EngineException] on error.
  Future<Task> startOONIRunV2MeasureDescriptorTask(
    OONIRunV2MeasureDescriptorConfig args,
  ) async {
    return _taskStart("OONIRunV2MeasureDescriptor", args);
  }

  /// Starts a new ExperimentMetaInfo task. Throws [EngineException] on error.
  Future<Task> startExperimentMetaInfoTask(
    ExperimentMetaInfoConfig args,
  ) async {
    return _taskStart("ExperimentMetaInfo", args);
  }

  /// Invokes OONITaskStart with the given task [name] and [args]. Throws
  /// [EngineException] in case we couldn't start a new task.
  Future<Task> _taskStart(String name, $pb.GeneratedMessage args) async {
    if (debug) {
      stderr.write("> TaskStart: ${name} ${args.toProto3Json()}\n");
    }
    final sender = await _getSender();
    sender.send(_TaskStartRequest(name, args));
    final r = await _receiver.next as _TaskStartResponse;
    if (debug) {
      stderr.write("< TaskStart: ${r.taskID}\n");
    }
    if (r.taskID < 0) {
      throw EngineException("Engine.taskStart: OONITaskStart failed");
    }
    return Task(this, r.taskID);
  }

  /// Invokes OONITaskWaitForNextEvent using the given [taskID] and
  /// [timeout], which is expressed in milliseconds.
  Future<$pb.GeneratedMessage?> taskWaitForNextEvent(
    int taskID,
    int timeout,
  ) async {
    if (taskID < 0) {
      return null; // short circuit
    }
    if (debug) {
      stderr.write("> TaskWaitForNextEvent: ${taskID} ${timeout}\n");
    }
    final sender = await _getSender();
    sender.send(_TaskWaitForNextEventRequest(taskID, timeout));
    final resp = await _receiver.next as _TaskWaitForNextEventResponse;
    if (debug) {
      stderr.write(
        "< TaskWaitForNextEvent: ${resp.event?.toProto3Json()}\n",
      );
    }
    return resp.event;
  }

  /// Invokes OONITaskIsDone for the given [taskID].
  Future<bool> taskIsDone(int taskID) async {
    if (taskID < 0) {
      return true; // short circuit
    }
    if (debug) {
      stderr.write("> TaskIsDone: ${taskID}\n");
    }
    final sender = await _getSender();
    sender.send(_TaskIsDoneRequest(taskID));
    final resp = await _receiver.next as _TaskIsDoneResponse;
    if (debug) {
      stderr.write("< TaskIsDone: ${resp.isDone}\n");
    }
    return resp.isDone;
  }

  /// Invokes OONITaskInterrupt for the given [taskID].
  void taskInterrupt(int taskID) async {
    if (taskID < 0) {
      return; // short circuit
    }
    if (debug) {
      stderr.write("> TaskInterrupt: ${taskID}\n");
    }
    final sender = await _getSender();
    sender.send(_TaskInterruptRequest(taskID));
  }

  /// Invokes OONITaskFree for the given [taskID].
  void taskFree(int taskID) async {
    if (!_initialized || taskID < 0) {
      return; // short circuit
    }
    if (debug) {
      stderr.write("> TaskFree: ${taskID}\n");
    }
    final sender = await _getSender();
    sender.send(_TaskFreeRequest(taskID));
  }

  /// Unregisters the reading port. You must call this function
  /// from your main or toplevel code to make Dart exit. It won't
  /// exit otherwise because it keeps reading from the port.
  void shutdown() async {
    if (!_initialized) {
      return; // idempotent
    }
    if (debug) {
      stderr.write("> Shutdown\n");
    }
    // Implementation note: sending a message to the service
    // to stop allows creating serveral services without
    // keeping their isolates around unnecessarily (I hope).
    final sender = await _getSender();
    sender.send(_ShutdownServiceRequest());
    await _receiver.next as _ShutdownServiceResponse;
    if (debug) {
      stderr.write("< Shutdown\n");
    }
    _receiver.cancel(immediate: true);
    _initialized = false;
  }
}

/// A running task.
class Task {
  /// The engine that created this task.
  final Engine _engine;

  /// The task's unique ID within the engine.
  int _id;

  /// Constructor.
  Task(this._engine, this._id);

  /// Invokes OONITaskWaitForNextEvent using this task's ID and
  /// [timeout], which is expressed in milliseconds.
  Future<$pb.GeneratedMessage?> waitForNextEvent(int timeout) async {
    return _engine.taskWaitForNextEvent(_id, timeout);
  }

  /// Invokes OONITaskIsDone for this task's ID.
  Future<bool> isDone() async {
    return _engine.taskIsDone(_id);
  }

  /// Invokes OONITaskInterrupt for this task's ID.
  void interrupt() async {
    _engine.taskInterrupt(_id);
  }

  /// Invokes OONITaskFree for this task's ID.
  void free() async {
    _engine.taskFree(_id);
    _id = -1;
  }

  /// Calls [pred] for each event emitted by this task until the task is done
  /// or [pred] returns true. We let exceptions unwind the stack.
  Future<void> foreachEvent(bool Function($pb.GeneratedMessage) pred) async {
    const timeout = 250; // millisecond
    while (!await isDone()) {
      final ev = await waitForNextEvent(timeout);
      if (ev == null) {
        continue; // hopefully just a timeout
      }
      if (pred(ev)) {
        break; // the user wants us to stop
      }
    }
  }

  /// Observes all the events emitted by this task until it sees an
  /// event of type [T]. When this happens, it returns to the caller.
  ///
  /// The [silent] argument controls whether we want this function
  /// to print all the encountered events until it finds a match.
  ///
  /// If an event with type [T] is not found, we return null.
  Future<T?> findFirst<T>({bool silent: true}) async {
    T? result = null;
    await foreachEvent((ev) {
      if (!silent) {
        printEvent(ev);
      }
      if (ev is T) {
        result = ev as T;
        return true;
      }
      return false;
    });
    return result;
  }

  /// Collects all the events with type [T] in the stream of events
  /// returned by this task until the task completes.
  ///
  /// The [silent] argument controls whether we want this function
  /// to print all the encountered events.
  Future<List<T>> collect<T>({bool silent: true}) async {
    List<T> out = [];
    await foreachEvent((ev) {
      if (!silent) {
        printEvent(ev);
      }
      if (ev is T) {
        out.add(ev as T);
      }
      return false;
    });
    return out;
  }
}

/// printEvent prints a human-readable representation of [ev]. If the [ev]
/// argument is null or unhandled, no message is printed.
///
/// This function always returns false such that it can be combined with
/// [Task.foreachEvent] to print all the events returned by a task.
bool printEvent($pb.GeneratedMessage? ev) {
  if (ev is LogEvent) {
    print("${ev.level}: ${ev.message}");
    return false;
  }

  if (ev is GeoIPEvent) {
    print("");
    print("GeoIP lookup result");
    print("-------------------");
    print("failure      : ${ev.failure == "" ? null : ev.failure}");
    print("probe_ip     : ${ev.probeIp}");
    print("probe_asn    : ${ev.probeAsn} (${ev.probeNetworkName})");
    print("probe_cc     : ${ev.probeCc}");
    print("resolver_ip  : ${ev.resolverIp}");
    print("resolver_asn : ${ev.resolverAsn} (${ev.resolverNetworkName})");
    print("");
    return false;
  }

  if (ev is ProgressEvent) {
    print("PROGRESS: ${ev.percentage * 100}%: ${ev.message}");
    return false;
  }

  if (ev is DataUsageEvent) {
    print(
      "DATA_USAGE: sent ${ev.kibiBytesSent} KiB recv ${ev.kibiBytesReceived} KiB",
    );
    return false;
  }

  if (ev is SubmitEvent) {
    print("SUBMIT: #${ev.index}... ${ev.failure == "" ? "ok" : ev.failure}");
    return false;
  }

  if (ev is ExperimentMetaInfoEvent) {
    print("Experiment: ${ev.name} usesInput=${ev.usesInput}");
    return false;
  }

  return false;
}
