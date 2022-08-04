// AUTO GENERATED FILE, DO NOT EDIT.

import 'dart:convert';

import 'abi.dart';
import 'engine.dart';

/// Allows you to run any task asynchronously.
class BaseTask {
  /// Reference to the OONI engine.
  final Engine _engine;

  /// The task name.
  final String _name;

  /// Reference to the task config.
  final BaseConfig _config;

  /// Reference to the task ID.
  int _taskID = -1;

  /// Whether we have started the task.
  bool _started = false;

  /// Construct instance using the given [engine], [name] and [config].
  BaseTask(this._engine, this._name, this._config);

  /// Starts task if needed and retrieves the next event. This method
  /// returns a null value when the task has terminated.
  Future<BaseEvent?> next() async {
    if (!_started) {
      _taskID = await _engine.taskStart(_name, jsonEncode(_config));
      if (_taskID < 0) {
        return null;
      }
      _started = true;
    }
    while (true) {
      final isDone = await _engine.taskIsDone(_taskID);
      if (isDone) {
        stop();
        return null;
      }
      const timeout = 250; // milliseconds
      final event = await _engine.taskWaitForNextEvent(_taskID, timeout);
      if (event == null) {
        continue;
      }
      final parsed = _parseEvent(event);
      if (parsed == null) {
        continue;
      }
      return parsed;
    }
  }

  /// Parses all the possible events. Returns null if the event
  /// name is not one of the registered events.
  BaseEvent? _parseEvent(Event ev) {
    if (ev.name == LogEventName) {
      Map<String, dynamic> val = jsonDecode(ev.value);
      return LogEventValue.fromJson(val);
    }
    if (ev.name == DataUsageEventName) {
      Map<String, dynamic> val = jsonDecode(ev.value);
      return DataUsageEventValue.fromJson(val);
    }
    if (ev.name == GeoIPEventName) {
      Map<String, dynamic> val = jsonDecode(ev.value);
      return GeoIPEventValue.fromJson(val);
    }
    if (ev.name == MetaInfoExperimentEventName) {
      Map<String, dynamic> val = jsonDecode(ev.value);
      return MetaInfoExperimentEventValue.fromJson(val);
    }
    if (ev.name == ProgressEventName) {
      Map<String, dynamic> val = jsonDecode(ev.value);
      return ProgressEventValue.fromJson(val);
    }
    if (ev.name == SubmitEventName) {
      Map<String, dynamic> val = jsonDecode(ev.value);
      return SubmitEventValue.fromJson(val);
    }
    return null;
  }

  /// Explicitly terminates the running task before
  /// it terminates naturally by interrupting it.
  void stop() {
    _engine.taskFree(_taskID);
    _taskID = -1;
    _started = false;
  }
}

/// Allows running the GeoIP task.
class GeoIPTask extends BaseTask {
  /// Construct instance using the given [engine] and [config].
  GeoIPTask(Engine engine, GeoIPConfig config) : super(engine, "GeoIP", config);
}

/// Allows running the MetaInfoExperiment task.
class MetaInfoExperimentTask extends BaseTask {
  /// Construct instance using the given [engine] and [config].
  MetaInfoExperimentTask(Engine engine, MetaInfoExperimentConfig config)
      : super(engine, "MetaInfoExperiment", config);
}

/// Allows running the Nettest task.
class NettestTask extends BaseTask {
  /// Construct instance using the given [engine] and [config].
  NettestTask(Engine engine, NettestConfig config)
      : super(engine, "Nettest", config);
}

/// Allows running the OONIRunV2MeasureDescriptor task.
class OONIRunV2MeasureDescriptorTask extends BaseTask {
  /// Construct instance using the given [engine] and [config].
  OONIRunV2MeasureDescriptorTask(
      Engine engine, OONIRunV2MeasureDescriptorConfig config)
      : super(engine, "OONIRunV2MeasureDescriptor", config);
}
