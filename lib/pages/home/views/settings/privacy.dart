import 'package:easy_localization/easy_localization.dart';
import 'package:flutter/material.dart';
import 'package:settings_ui/settings_ui.dart';

class Privacy extends StatelessWidget {
  const Privacy({
    Key? key,
  }) : super(key: key);

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: const Text('Settings.Privacy.Label').tr(),
      ),
      body: SettingsList(sections: [
        SettingsSection(
          tiles: [
            SettingsTile.switchTile(
              onToggle: (value) {},
              initialValue: true,
              title: const Text('Settings.Sharing.UploadResults').tr(),
            ),
            SettingsTile.switchTile(
              onToggle: (value) {},
              initialValue: true,
              title: const Text('Settings.Privacy.SendCrashReports').tr(),
            ),
          ],
        )
      ]),
    );
  }
}
