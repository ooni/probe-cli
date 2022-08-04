/// Syntactic sugar to perform common operations.

import 'package:ooniengine/abi.dart';
import 'package:ooniengine/tasks.dart';

/// Collects all the events with type T in the stream of events
/// returned by [task] and stops the task when done.
///
/// This function will additionally log any [LogEventValue] event
/// unless [T] is [LogEventValue], in which case it will just return
/// all the emitted [LogEventValue] events to the caller.
///
/// In case of exception, this function will log the exception
/// and then return an empty list to the caller.
Future<List<T>> collect<T>(BaseTask task) async {
  List<T> out = [];
  await foreachEvent(task, (ev) {
    if (ev is T) {
      out.add(ev as T);
    }
    return printEvent(ev);
  });
  return out;
}

/// Observes all the events emitted by [task] until it sees an
/// event of type [T]. When this happens, it stops the [task] and
/// returns the collected event to the caller.
///
/// This function will print [LogEventValue] events unless [T]
/// is [LogEventValue], in which case it will return the first such
/// event to the caller without printing anything.
///
/// If an event with type [T] is not found or there is an
/// exception, this function returns null.
Future<T?> findFirst<T>(BaseTask task) async {
  T? result = null;
  await foreachEvent(task, (ev) {
    if (ev is T) {
      result = ev as T;
      return true;
    }
    if (ev is LogEventValue) {
      return printEvent(ev);
    }
    return false;
  });
  return result;
}

/// Calls [pred] for each event emitted by [task] until. Stops when
/// [task] is done or an exception occcurs or [pred] returns true.
Future<void> foreachEvent(BaseTask task, bool Function(BaseEvent) pred) async {
  try {
    while (true) {
      final ev = await task.next();
      if (ev == null) {
        break; // we have read all events
      }
      if (pred(ev)) {
        break; // the user wants us to stop earlier
      }
    }
  } catch (exc) {
    print("EXCEPTION: ${exc}");
    // suppress the exception after printing it
  } finally {
    task.stop();
  }
  return;
}

/// printEvent prints the canonical string representation of
/// the given [ev]. This function always returns false so you can
/// combine it with [forachEvent] to print all events. If the
/// [ev] argument is null or unhandled, this function does not
/// print any message on the standard output.
bool printEvent(BaseEvent? ev) {
  if (ev == null) {
    return false;
  }

  if (ev is LogEventValue) {
    print("${ev.level}: ${ev.message}");
    return false;
  }

  if (ev is GeoIPEventValue) {
    print("");
    print("GeoIP lookup result");
    print("-------------------");
    print("failure      : ${ev.failure == "" ? null : ev.failure}");
    print("probe_ip     : ${ev.probeIP}");
    print("probe_asn    : ${ev.probeASN} (${ev.probeNetworkName})");
    print("probe_cc     : ${ev.probeCC}");
    print("resolver_ip  : ${ev.resolverIP}");
    print("resolver_asn : ${ev.resolverASN} (${ev.resolverNetworkName})");
    print("");
    return false;
  }

  if (ev is ProgressEventValue) {
    print("PROGRESS: ${ev.percentage * 100}%: ${ev.message}");
    return false;
  }

  if (ev is SubmitEventValue) {
    print("SUBMIT: #${ev.index}... ${ev.failure == "" ? "ok" : ev.failure}");
    return false;
  }

  return false;
}
