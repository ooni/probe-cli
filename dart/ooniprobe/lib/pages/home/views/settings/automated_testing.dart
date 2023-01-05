import 'package:easy_localization/easy_localization.dart';
import 'package:flutter/material.dart';
import 'package:settings_ui/settings_ui.dart';

class AutomatedTesting extends StatelessWidget {
  const AutomatedTesting({
    Key? key,
  }) : super(key: key);

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title:
        const Text('Settings.AutomatedTesting.Label')
            .tr(),
      ),
      body: SettingsList(
        platform: DevicePlatform.android,
        sections: [
          SettingsSection(
            tiles: [
              SettingsTile.switchTile(
                onToggle: (value) {},
                initialValue: true,
                title: const Text(
                    'Settings.AutomatedTesting.RunAutomatically')
                    .tr(),
                description: Column(
                  mainAxisAlignment:
                  MainAxisAlignment.start,
                  crossAxisAlignment:
                  CrossAxisAlignment.start,
                  children: [
                    const Text(
                        'Settings.AutomatedTesting.RunAutomatically.Number')
                        .tr(),
                    const Text(
                        'Settings.AutomatedTesting.RunAutomatically.DateLast')
                        .tr(),
                  ],
                ),
              ),
              SettingsTile.switchTile(
                onToggle: (value) {},
                initialValue: true,
                title: const Text(
                    'Settings.AutomatedTesting.RunAutomatically.WiFiOnly')
                    .tr(),
              ),
              SettingsTile.switchTile(
                onToggle: (value) {},
                initialValue: true,
                title: const Text(
                    'Settings.AutomatedTesting.RunAutomatically.ChargingOnly')
                    .tr(),
              ),
              CustomSettingsTile(
                child: Padding(
                  padding: const EdgeInsets.symmetric(
                      horizontal: 20),
                  child: const Text(
                      'Settings.AutomatedTesting.RunAutomatically.Footer')
                      .tr(),
                ),
              )
            ],
          )
        ],
      ),
    );
  }
}

