import 'package:flutter/material.dart';
import 'package:ooniengine/abi.pb.dart';

class RunDescriptor {
  RunDescriptor._({
    required this.runV2Descriptor,
    required this.methodology,
    required this.icon,
  }) : super();

  final OONIRunV2Descriptor runV2Descriptor;
  final String methodology;
  final IconData icon;
  factory RunDescriptor({
    required String name,
    required String description,
    required String author,
    required String methodology,
    required IconData icon,
    required List<OONIRunV2DescriptorNettest> nettests,
  }) {
    return RunDescriptor._(
      runV2Descriptor: OONIRunV2Descriptor(
        name: name,
        description: description,
        author: author,
        nettests: nettests,
      ),
      methodology: methodology,
      icon: icon,
    );
  }

  String get name {
    return runV2Descriptor.name;
  }

  String get description {
    return runV2Descriptor.description;
  }

  String get author {
    return runV2Descriptor.author;
  }

  List<OONIRunV2DescriptorNettest> get nettests {
    return runV2Descriptor.nettests;
  }
}
