package main

//
// Dart code generator
//

import (
	"io"

	"github.com/iancoleman/strcase"
)

// dartType maps a given Type to an actual dart type.
func dartType(kind Type) string {
	switch kind {
	case TypeString:
		return "String"
	case TypeInt64:
		return "int"
	case TypeListString:
		return "List<String>"
	case TypeMapStringString:
		return "Map<String, String>"
	case TypeMapStringAny:
		return "Map<String, Object>"
	case TypeBool:
		return "bool"
	case TypeFloat64:
		return "double"
	case OONIRunV2NettestListType:
		return "List<OONIRunV2Nettest>" // special case
	default:
		return string(kind)
	}
}

// dartFieldName generates the correct dart field name.
func dartFieldName(name string) string {
	return strcase.ToLowerCamel(name)
}

// generateABIDart generates the abi.dart file.
func generateABIDart(fp io.Writer, abiVersion string, abi *ABI) {
	writeFile(fp, "// AUTO GENERATED FILE, DO NOT EDIT.\n")
	writeFile(fp, "\n")

	writeFile(fp, "import 'package:json_annotation/json_annotation.dart';\n")
	writeFile(fp, "\n")

	writeFile(fp, "part 'abi.g.dart';\n")
	writeFile(fp, "\n")

	writeFile(fp, "//\n")
	writeFile(fp, "// Auto-generated ABI.\n")
	writeFile(fp, "//\n")
	writeFile(fp, "\n")

	writeFile(fp, "/// ABI version number.\n")
	writeFile(fp, "const ABIVersion = \"%s\";\n", abiVersion)
	writeFile(fp, "\n")

	for index, constant := range abi.Constants {
		for _, doc := range constant.Docs {
			writeFile(fp, "/// %s\n", doc)
		}
		writeFile(fp, "const %s = \"%s\";\n", constant.Name, constant.Value)
		if index != len(abi.Constants)-1 {
			writeFile(fp, "\n")
		}
	}
	writeFile(fp, "\n")

	for _, structdef := range abi.Structs {
		for _, doc := range structdef.Docs {
			writeFile(fp, "/// %s\n", doc)
		}
		writeFile(fp, "@JsonSerializable()\n")
		if structdef.Superclass != "" {
			writeFile(fp, "class %s extends %s {\n", structdef.Name, structdef.Superclass)
		} else {
			writeFile(fp, "class %s {\n", structdef.Name)
		}
		for index, field := range structdef.Fields {
			for _, doc := range field.Docs {
				writeFile(fp, "  /// %s\n", doc)
			}
			name := dartFieldName(field.Name)
			kind := dartType(field.Type)
			tag := strcase.ToSnake(field.Name)
			writeFile(fp, "  @JsonKey(name: \"%s\")\n", tag)
			writeFile(fp, "  final %s %s;\n", kind, name)
			if index != len(structdef.Fields)-1 {
				writeFile(fp, "\n")
			}
		}

		writeFile(fp, "\n")
		writeFile(fp, "  /// Default constructor.\n")
		if len(structdef.Fields) > 0 {
			writeFile(fp, "  %s({\n", structdef.Name)
			for _, field := range structdef.Fields {
				name := dartFieldName(field.Name)
				writeFile(fp, "    required this.%s,\n", name)
			}
			writeFile(fp, "  });\n")
		} else {
			writeFile(fp, "  %s();\n", structdef.Name)
		}

		writeFile(fp, "\n")
		writeFile(fp, "  /// Factory to construct from JSON.\n")
		writeFile(
			fp,
			"  factory %s.fromJson(Map<String, dynamic> json) => _$%sFromJson(json);\n",
			structdef.Name,
			structdef.Name,
		)

		writeFile(fp, "\n")
		writeFile(fp, "  /// Serialize to JSON.\n")
		writeFile(
			fp,
			"  Map<String, dynamic> toJson() => _$%sToJson(this);\n",
			structdef.Name,
		)

		writeFile(fp, "}\n")
		writeFile(fp, "\n")
	}
}

// generateTasksDart generates the tasks.dart file.
func generateTasksDart(fp io.Writer, abi *ABI) {
	writeFile(fp, "// AUTO GENERATED FILE, DO NOT EDIT.\n")
	writeFile(fp, "\n")

	writeFile(fp, "import 'dart:convert';\n")
	writeFile(fp, "\n")

	writeFile(fp, "import 'abi.dart';\n")
	writeFile(fp, "import 'engine.dart';\n")
	writeFile(fp, "\n")

	writeFile(fp, "/// Allows you to run any task asynchronously.\n")
	writeFile(fp, "class BaseTask {\n")
	writeFile(fp, "  /// Reference to the OONI engine.\n")
	writeFile(fp, "  final Engine _engine;\n")
	writeFile(fp, "\n")

	writeFile(fp, "  /// The task name.\n")
	writeFile(fp, "  final String _name;\n")
	writeFile(fp, "\n")

	writeFile(fp, "  /// Reference to the task config.\n")
	writeFile(fp, "  final BaseConfig _config;\n")
	writeFile(fp, "\n")

	writeFile(fp, "  /// Reference to the task ID.\n")
	writeFile(fp, "  int _taskID = -1;\n")
	writeFile(fp, "\n")

	writeFile(fp, "  /// Whether we have started the task.\n")
	writeFile(fp, "  bool _started = false;\n")
	writeFile(fp, "\n")

	writeFile(fp, "  /// Construct instance using the given [engine], [name] and [config].\n")
	writeFile(fp, "  BaseTask(this._engine, this._name, this._config);\n")
	writeFile(fp, "\n")

	writeFile(fp, "  /// Starts task if needed and retrieves the next event. This method\n")
	writeFile(fp, "  /// returns a null value when the task has terminated.\n")
	writeFile(fp, "  Future<BaseEvent?> next() async {\n")
	writeFile(fp, "    if (!_started) {\n")
	writeFile(fp, "      _taskID = await _engine.taskStart(_name, jsonEncode(_config));\n")
	writeFile(fp, "      if (_taskID < 0) {\n")
	writeFile(fp, "        return null;\n")
	writeFile(fp, "      }\n")
	writeFile(fp, "      _started = true;\n")
	writeFile(fp, "    }\n")
	writeFile(fp, "    while (true) {\n")
	writeFile(fp, "      final isDone = await _engine.taskIsDone(_taskID);\n")
	writeFile(fp, "      if (isDone) {\n")
	writeFile(fp, "        stop();\n")
	writeFile(fp, "        return null;\n")
	writeFile(fp, "      }\n")
	writeFile(fp, "      const timeout = 250; // milliseconds\n")
	writeFile(fp, "      final event = await _engine.taskWaitForNextEvent(_taskID, timeout);\n")
	writeFile(fp, "      if (event == null) {\n")
	writeFile(fp, "        continue;\n")
	writeFile(fp, "      }\n")
	writeFile(fp, "      final parsed =  _parseEvent(event);\n")
	writeFile(fp, "      if (parsed == null) {\n")
	writeFile(fp, "        continue;\n")
	writeFile(fp, "      }\n")
	writeFile(fp, "      return parsed;\n")
	writeFile(fp, "    }\n")
	writeFile(fp, "  }\n")
	writeFile(fp, "\n")

	writeFile(fp, "  /// Parses all the possible events. Returns null if the event\n")
	writeFile(fp, "  /// name is not one of the registered events.\n")
	writeFile(fp, "  BaseEvent? _parseEvent(Event ev) {\n")
	for _, structdef := range abi.Structs {
		if !structdef.isEvent() {
			continue
		}
		name := structdef.toEventName() + EventNameSuffix
		writeFile(fp, "    if (ev.name == %s) {\n", name)
		writeFile(fp, "      Map<String, dynamic> val = jsonDecode(ev.value);\n")
		writeFile(fp, "      return %s.fromJson(val);\n", structdef.Name)
		writeFile(fp, "    }\n")
	}
	writeFile(fp, "    return null;\n")
	writeFile(fp, "  }\n")
	writeFile(fp, "\n")

	writeFile(fp, "  /// Explicitly terminates the running task before\n")
	writeFile(fp, "  /// it terminates naturally by interrupting it.\n")
	writeFile(fp, "  void stop() {\n")
	writeFile(fp, "    _engine.taskFree(_taskID);\n")
	writeFile(fp, "    _taskID = -1;\n")
	writeFile(fp, "    _started = false;\n")
	writeFile(fp, "  }\n")
	writeFile(fp, "}\n")
	writeFile(fp, "\n")

	for _, task := range abi.Tasks {
		writeFile(fp, "/// Allows running the %s task.\n", task.Name)
		writeFile(fp, "class %sTask extends BaseTask {\n", task.Name)

		writeFile(fp, "  /// Construct instance using the given [engine] and [config].\n")
		writeFile(fp, "  %sTask(Engine engine, %sConfig config)\n", task.Name, task.Name)
		writeFile(fp, "      : super(engine, \"%s\", config);\n", task.Name)
		writeFile(fp, "\n")
		writeFile(fp, "}\n")
		writeFile(fp, "\n")
	}
}
