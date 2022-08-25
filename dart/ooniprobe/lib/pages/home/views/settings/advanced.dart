import 'package:easy_localization/easy_localization.dart';
import 'package:flutter/material.dart';
import 'package:settings_ui/settings_ui.dart';

class Advanced extends StatelessWidget {
  const Advanced({
    Key? key,
  }) : super(key: key);

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: const Text('Settings.Advanced.Label').tr(),
      ),
      body: SettingsList(
        platform: DevicePlatform.android,
        sections: [
          SettingsSection(
            tiles: [
              CustomSettingsTile(
                child: Padding(
                  padding: const EdgeInsets.symmetric(horizontal: 24,vertical: 10),
                  child: Row(
                    mainAxisAlignment: MainAxisAlignment.spaceBetween,
                    children: [
                      Column(
                        crossAxisAlignment: CrossAxisAlignment.start,
                        children: [
                          DefaultTextStyle(
                            style: const TextStyle(
                              color: Color.fromARGB(255, 27, 27, 27),
                              fontSize: 18,
                              fontWeight: FontWeight.w400,
                            ),
                            child: const Text(
                                    'Settings.Advanced.LanguageSettings.Title')
                                .tr(),
                          ),
                          const Text('en').tr()
                        ],
                      ),
                      TextButton(
                        child: const Text(
                                'Settings.Advanced.LanguageSettings.PopUp')
                            .tr(),
                        onPressed: () {},
                      ),
                    ],
                  ),
                ),
              ),
              SettingsTile.switchTile(
                onToggle: (value) {},
                initialValue: true,
                title: const Text('Settings.Advanced.DebugLogs').tr(),
              ),
              CustomSettingsTile(
                child: Padding(
                  padding: const EdgeInsets.symmetric(horizontal: 24,vertical: 10),
                  child: Row(
                    mainAxisAlignment: MainAxisAlignment.spaceBetween,
                    children: [
                      Column(
                        crossAxisAlignment: CrossAxisAlignment.start,
                        children: [
                          DefaultTextStyle(
                            style: const TextStyle(
                              color: Color.fromARGB(255, 27, 27, 27),
                              fontSize: 18,
                              fontWeight: FontWeight.w400,
                            ),
                            child: const Text(
                                'Settings.Storage.Usage.Label')
                                .tr(),
                          ),
                          const Text('x kb').tr()
                        ],
                      ),
                      TextButton(
                        child: const Text(
                            'Settings.Storage.Clear')
                            .tr(),
                        onPressed: () {},
                      ),
                    ],
                  ),
                ),
              ),
              SettingsTile.switchTile(
                onToggle: (value) {},
                initialValue: true,
                title: const Text('Settings.WarmVPNInUse.Label').tr(),
              ),
            ],
          )
        ],
      ),
    );
  }
}
