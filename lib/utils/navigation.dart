import 'package:flutter/material.dart';

class Navigation {
  static Future<T?> navigateTo<T>({
    required BuildContext context,
    required Widget screen,
  }) async {
    return await Navigator.push<T>(
        context, MaterialPageRoute<T>(builder: (_) => screen));
  }
}
