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

  /// Converts [name] and [msg] to a Pointer<[OONIMessage]>.
  ///
  /// Returns nullptr in case of failure.
  Pointer<OONIMessage> _toNativeMessage(String name, $pb.GeneratedMessage msg) {
    Pointer<Uint8> nbase = nullptr;
    Pointer<Char> nname = nullptr;
    Pointer<OONIMessage> nargs = nullptr;
    try {
      nname = name.toNativeUtf8().cast<Char>(); // 🤓 needs free
      final writer = $pb.CodedBufferWriter();
      msg.writeToCodedBufferWriter(writer);
      final len = writer.lengthInBytes;
      nbase = malloc<Uint8>(len); // 🤓 needs free
      final memview = nbase.asTypedList(len);
      writer.writeTo(memview); // 🤓 first copy (Go will copy again)
      // struct OONIMessage {
      //   char     *Key;
      //   uint8_t  *Base;
      //   uint32_t Size;
      // };
      nargs = malloc<OONIMessage>(); // 🤓 needs free
      nargs.ref.Key = nname;
      nargs.ref.Base = nbase;
      if (len < 0 || len > UINT32_MAX) {
        throw Exception("len's value is < 0 or > UINT32_MAX");
      }
      nargs.ref.Size = len;
    } catch (exc) {
      malloc.free(nbase);
      malloc.free(nname);
      malloc.free(nargs);
      return nullptr;
    }
    return nargs;
  }

  /// Frees the memory used by a fully allocated Pointer<[OONIMessage]>.
  void _freeNativeMessage(Pointer<OONIMessage> msg) {
    if (msg != nullptr) {
      malloc.free(msg.ref.Base);
      malloc.free(msg.ref.Key);
    }
    malloc.free(msg);
  }

  /// Wrapper for OONITaskStart. The [name] argument contains the
  /// task name. The [args] argument contains the corresponding
  /// protobuf message containing the startup arguments for the task.
  ///
  /// A zero return value indicates failure. In such a case,
  /// OONITaskStart prints diagnostic messages on stderr.
  ///
  /// We let exceptions unwind the stack. In case of exceptions,
  /// this function does not leak any heap allocated memory.
  int taskStart(String name, $pb.GeneratedMessage args) {
    Pointer<OONIMessage> msg = nullptr;
    int taskHandle = 0;
    try {
      msg = _toNativeMessage(name, args); // 🤓 we alloc
      if (msg != nullptr) {
        // uintptr_t OONITaskStart(struct OONIMessage *msg);
        taskHandle = _engine.OONITaskStart(msg);
      }
    } finally {
      _freeNativeMessage(msg); // 🤓 we free
    }
    return taskHandle;
  }

  /// OONITaskWaitForNextEvent wrapper. The [taskHandle] argument is
  /// the opaque task handle returned by OONITaskStart. The
  /// [timeout] argument contains the timeout in milliseconds. Use
  /// a zero or negative [timeout] to block until there's a new event ready.
  ///
  /// A null return value indicates a timeout, that the task is done, or
  /// any unlikely OONITaskWaitForNextEvent internal error.
  ///
  /// To know whether a task is actually done use OONITaskIsDone.
  ///
  /// We let exceptions unwind the stack but we also ensure
  /// we don't leak any memory if there's an exception.
  $pb.GeneratedMessage? taskWaitForNextEvent(int taskHandle, int timeout) {
    if (timeout < 0) {
      timeout = 0; // normalize
    }
    if (timeout > INT32_MAX) {
      timeout = INT32_MAX; // likewise
    }
    // struct OONIMessage *OONITaskWaitForNextEvent(uintptr_t, int32_t);
    final ev = _engine.OONITaskWaitForNextEvent(taskHandle, timeout);
    return _ownAndConverFromNative(ev); // 🤓 takes ownership
  }

  /// Takes in input a native Pointer<[OONIMessage]> and parses it.
  ///
  /// This function TAKES OWNERSHIP of its argument and frees it when
  /// it is done processing it.
  $pb.GeneratedMessage? _ownAndConverFromNative(Pointer<OONIMessage> msg) {
    $pb.GeneratedMessage? out = null;
    try {
      if (msg != nullptr && msg.ref.Size >= 1) {
        // struct OONIMessage {
        //   char     *Key;
        //   uint8_t  *Base;
        //   uint32_t Size;
        // };
        final name = msg.ref.Key.cast<Utf8>().toDartString(); // 🤓 small
        final len = msg.ref.Size;
        final mv = msg.ref.Base.cast<Uint8>().asTypedList(len); // 🤓 zero copy
        out = _parseMessage(name, mv);
      }
    } finally {
      _engine.OONIMessageFree(msg); // 🤓 API is robust to nullptr
    }
    return out;
  }

  /// Parses from [data] the protobuf message corresponding to [name].
  $pb.GeneratedMessage? _parseMessage(String name, Uint8List data) {
    switch (name) {
      case "GeoIP":
        return GeoIPEvent.fromBuffer(data);
      case "Log":
        return LogEvent.fromBuffer(data);
      case "Progress":
        return ProgressEvent.fromBuffer(data);
      case "DataUsage":
        return DataUsageEvent.fromBuffer(data);
      case "Submit":
        return SubmitEvent.fromBuffer(data);
      case "ExperimentMetaInfoResponse":
        return ExperimentMetaInfoResponse.fromBuffer(data);
      default:
        stderr.write("LLEngine._parseMessage: unhandled name: ${name}\n");
        return null;
    }
  }

  /// Calls OONICall and returns the response.
  $pb.GeneratedMessage? call(String name, $pb.GeneratedMessage args) {
    Pointer<OONIMessage> req = nullptr;
    Pointer<OONIMessage> libResp = nullptr;
    $pb.GeneratedMessage? resp = null;
    try {
      req = _toNativeMessage(name, args); // 🤓 we alloc
      if (req != nullptr) {
        // struct OONIMessage *OONICall(struct OONIMessage *req);
        libResp = _engine.OONICall(req);
        resp = _ownAndConverFromNative(libResp); // 🤓 transfer ownership
      }
    } finally {
      _freeNativeMessage(req); // 🤓 we free
    }
    return resp;
  }

  /// Calls OONITaskIsDone to determine whether the task with [taskHandle]
  /// is done. A task is done when (1) it has stopped and (2) we've read
  /// all the emitted events using OONITaskWaitForNextEvent.
  bool taskIsDone(int taskHandle) {
    return _engine.OONITaskIsDone(taskHandle) != 0;
  }

  /// Calls OONITaskInterrupt to interrupt the task with [taskHandle]. An
  /// interrupted task will try to stop running ASAP.
  void taskInterrupt(int taskHandle) {
    _engine.OONITaskInterrupt(taskHandle);
  }

  /// Calls OONITaskFree to forget about the task with [taskHandle]. The
  /// OONI engine will internally interrupt the task and drain its
  /// events queue for us. That is, a task might continue to run for
  /// some time inside the engine after you call this method.
  void taskFree(int taskHandle) {
    _engine.OONITaskFree(taskHandle);
  }
}

/// An [LLEngine] instance running as a service inside an Isolate.
class IsolatedEngine {
  // Implementation note: we MUST open the DLL in its own isolate because
  // we cannot transfer it to another isolate.

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
    if (msg is _CallRequest) {
      final resp = engine.call(msg.name, msg.args);
      _sender.send(_CallResponse(resp));
      return false;
    }
    if (msg is _TaskStartRequest) {
      final task = engine.taskStart(msg.name, msg.args);
      _sender.send(_TaskStartResponse(task));
      return false;
    }
    if (msg is _TaskWaitForNextEventRequest) {
      final event = engine.taskWaitForNextEvent(msg.taskHandle, msg.timeout);
      _sender.send(_TaskWaitForNextEventResponse(event));
      return false;
    }
    if (msg is _TaskIsDoneRequest) {
      final isDone = engine.taskIsDone(msg.taskHandle);
      _sender.send(_TaskIsDoneResponse(isDone));
      return false;
    }
    if (msg is _TaskInterruptRequest) {
      engine.taskInterrupt(msg.taskHandle);
      return false;
    }
    if (msg is _TaskFreeRequest) {
      engine.taskFree(msg.taskHandle);
      return false;
    }
    if (msg is _ShutdownServiceRequest) {
      _sender.send(_ShutdownServiceResponse());
      return true;
    }
    return false;
  }
}

/// Request to call a method.
class _CallRequest {
  /// Method name.
  final String name;

  /// Method arguments.
  final $pb.GeneratedMessage args;

  /// Creates a new instance.
  _CallRequest(this.name, this.args);
}

/// Result of calling a method.
class _CallResponse {
  /// Optional result (a null event implies internal error).
  final $pb.GeneratedMessage? resp;

  /// Creates a new instance.
  _CallResponse(this.resp);
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
  /// Opaque handle of the task.
  final int taskHandle;

  /// Creates a new instance.
  _TaskStartResponse(this.taskHandle);
}

/// Request to wait for the next task event.
class _TaskWaitForNextEventRequest {
  /// Opaque handle of the task.
  final int taskHandle;

  /// Wait timeout in milliseconds.
  final int timeout;

  /// Creates a new instance.
  _TaskWaitForNextEventRequest(this.taskHandle, this.timeout);
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
  /// Opaque handle of the task.
  final int taskHandle;

  /// Creates a new instance.
  _TaskIsDoneRequest(this.taskHandle);
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
  /// Opaque handle of the task.
  final int taskHandle;

  /// Creates a new instance.
  _TaskInterruptRequest(this.taskHandle);
}

/// Request to free all memory associated with a task.
class _TaskFreeRequest {
  /// Opaque handle of the task.
  final int taskHandle;

  /// Creates a new instance.
  _TaskFreeRequest(this.taskHandle);
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

  /// Converts to string.
  String toString() {
    return cause;
  }
}

/// Helps to assign a unique ID to each engine.
var _uniqueEngineID = 0;

/// Generates the next unique engine ID.
int _nextUniqueEngineID() {
  _uniqueEngineID++;
  return _uniqueEngineID;
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
///
/// This struct will log messages exchanged with the engine
/// on stderr if `export OONI_ENGINE_DEBUG=1`.
class Engine {
  /// Unique engine ID.
  final _id = _nextUniqueEngineID();

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
  bool _debug = Platform.environment["OONI_ENGINE_DEBUG"] == "1";

  /// Constructs using the given library path, which must be the
  /// absolute path to the OONIEngine DLL.
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

  /// Calls the experimentMetaInfo method.
  Future<ExperimentMetaInfoResponse> callExperimentMetaInfo() async {
    final r = await _call("ExperimentMetaInfo", ExperimentMetaInfoRequest());
    if (r == null) {
      throw EngineException("ExperimentMetaInfo failed");
    }
    return r as ExperimentMetaInfoResponse;
  }

  /// Calls a remote method and returns a response.
  Future<$pb.GeneratedMessage?> _call(
    String name,
    $pb.GeneratedMessage req,
  ) async {
    if (_debug) {
      stderr.write("[#${_id}] > Call: ${name} ${req.toProto3Json()}\n");
    }
    final sender = await _getSender();
    sender.send(_CallRequest(name, req));
    final r = await _receiver.next as _CallResponse;
    if (_debug) {
      stderr.write("[#${_id}] < Call: ${r.resp?.toProto3Json()}\n");
    }
    return r.resp;
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

  /// Invokes OONITaskStart with the given task [name] and [args]. Throws
  /// [EngineException] in case we couldn't start a new task.
  Future<Task> _taskStart(String name, $pb.GeneratedMessage args) async {
    if (_debug) {
      stderr.write("[#${_id}] > TaskStart: ${name} ${args.toProto3Json()}\n");
    }
    final sender = await _getSender();
    sender.send(_TaskStartRequest(name, args));
    final r = await _receiver.next as _TaskStartResponse;
    if (_debug) {
      stderr.write("[#${_id}] < TaskStart: handle=${r.taskHandle}\n");
    }
    if (r.taskHandle == 0) {
      throw EngineException("Engine.taskStart: OONITaskStart failed");
    }
    return Task(this, r.taskHandle);
  }

  /// Invokes OONITaskWaitForNextEvent using the given [taskHandle] and
  /// [timeout], which is expressed in milliseconds.
  Future<$pb.GeneratedMessage?> taskWaitForNextEvent(
    int taskHandle,
    int timeout,
  ) async {
    if (taskHandle == 0) {
      return null; // short circuit
    }
    if (_debug) {
      stderr.write(
        "[#${_id}] > TaskWaitForNextEvent: handle=${taskHandle} timeout=${timeout}\n",
      );
    }
    final sender = await _getSender();
    sender.send(_TaskWaitForNextEventRequest(taskHandle, timeout));
    final resp = await _receiver.next as _TaskWaitForNextEventResponse;
    if (_debug) {
      stderr.write(
        "[#${_id}] < TaskWaitForNextEvent: ${resp.event?.toProto3Json()}\n",
      );
    }
    return resp.event;
  }

  /// Invokes OONITaskIsDone for the given [taskHandle].
  Future<bool> taskIsDone(int taskHandle) async {
    if (taskHandle == 0) {
      return true; // short circuit
    }
    if (_debug) {
      stderr.write("[#${_id}] > TaskIsDone: handle=${taskHandle}\n");
    }
    final sender = await _getSender();
    sender.send(_TaskIsDoneRequest(taskHandle));
    final resp = await _receiver.next as _TaskIsDoneResponse;
    if (_debug) {
      stderr.write("[#${_id}] < TaskIsDone: ${resp.isDone}\n");
    }
    return resp.isDone;
  }

  /// Invokes OONITaskInterrupt for the given [taskHandle].
  void taskInterrupt(int taskHandle) async {
    if (taskHandle == 0) {
      return; // short circuit
    }
    if (_debug) {
      stderr.write("[#${_id}] > TaskInterrupt: handle=${taskHandle}\n");
    }
    final sender = await _getSender();
    sender.send(_TaskInterruptRequest(taskHandle));
  }

  /// Invokes OONITaskFree for the given [taskHandle].
  void taskFree(int taskHandle) async {
    if (!_initialized || taskHandle == 0) {
      return; // short circuit
    }
    if (_debug) {
      stderr.write("[#${_id}] > TaskFree: handle=${taskHandle}\n");
    }
    final sender = await _getSender();
    sender.send(_TaskFreeRequest(taskHandle));
  }

  /// Unregisters the reading port. You must call this function
  /// from your main or toplevel code to make Dart exit. It won't
  /// exit otherwise because it keeps reading from the port.
  void shutdown() async {
    if (!_initialized) {
      return; // idempotent
    }
    if (_debug) {
      stderr.write("[#${_id}] > Shutdown\n");
    }
    // Implementation note: sending a message to the service
    // to stop allows creating serveral services without
    // keeping their isolates around unnecessarily (I hope).
    final sender = await _getSender();
    sender.send(_ShutdownServiceRequest());
    await _receiver.next as _ShutdownServiceResponse;
    if (_debug) {
      stderr.write("[#${_id}] < Shutdown\n");
    }
    _receiver.cancel(immediate: true);
    _initialized = false;
  }
}

/// A running task.
class Task {
  /// The engine that created this task.
  final Engine _engine;

  /// The task's opaque handle.
  int _handle;

  /// Constructor.
  Task(this._engine, this._handle);

  /// Invokes OONITaskWaitForNextEvent using this task's ID and
  /// [timeout], which is expressed in milliseconds.
  Future<$pb.GeneratedMessage?> waitForNextEvent(int timeout) async {
    return _engine.taskWaitForNextEvent(_handle, timeout);
  }

  /// Invokes OONITaskIsDone for this task's ID.
  Future<bool> isDone() async {
    return _engine.taskIsDone(_handle);
  }

  /// Invokes OONITaskInterrupt for this task's ID.
  void interrupt() async {
    _engine.taskInterrupt(_handle);
  }

  /// Invokes OONITaskFree for this task's ID.
  void free() async {
    _engine.taskFree(_handle);
    _handle = 0; // short circuit subsequent calls
  }

  /// The timeout used when calling waitForNextEvent (in millisecond).
  ///
  /// A timeout ensures we do not block the isolate's event loop, so it can
  /// read subsequent commands sent by us (e.g., an interrupt).
  ///
  /// For this reason, it's not recommended to set this value to a zero
  /// or negative value, which will disable timing out.
  int timeout = 250;

  /// Calls [pred] for each event emitted by this task until the task is done
  /// or [pred] returns true. We let exceptions unwind the stack.
  Future<void> foreachEvent(bool Function($pb.GeneratedMessage) pred) async {
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
        printMessage(ev);
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
        printMessage(ev);
      }
      if (ev is T) {
        out.add(ev as T);
      }
      return false;
    });
    return out;
  }
}

/// printEvent prints a human-readable representation of [msg]. If the [msg]
/// argument is null or unhandled, no message is printed.
///
/// This function always returns false such that it can be combined with
/// [Task.foreachEvent] to print all the events returned by a task.
bool printMessage($pb.GeneratedMessage? msg) {
  if (msg is LogEvent) {
    print("${msg.level}: ${msg.message}");
    return false;
  }

  if (msg is GeoIPEvent) {
    print("");
    print("GeoIP lookup result");
    print("-------------------");
    print("failure      : ${msg.failure == "" ? null : msg.failure}");
    print("probe_ip     : ${msg.probeIp}");
    print("probe_asn    : ${msg.probeAsn} (${msg.probeNetworkName})");
    print("probe_cc     : ${msg.probeCc}");
    print("resolver_ip  : ${msg.resolverIp}");
    print("resolver_asn : ${msg.resolverAsn} (${msg.resolverNetworkName})");
    print("");
    return false;
  }

  if (msg is ProgressEvent) {
    print("PROGRESS: ${msg.percentage * 100}%: ${msg.message}");
    return false;
  }

  if (msg is DataUsageEvent) {
    print(
      "DATA_USAGE: sent ${msg.kibiBytesSent} KiB recv ${msg.kibiBytesReceived} KiB",
    );
    return false;
  }

  if (msg is SubmitEvent) {
    print("SUBMIT: #${msg.index}... ${msg.failure == "" ? "ok" : msg.failure}");
    return false;
  }

  // We don't print unknown messages.
  return false;
}
