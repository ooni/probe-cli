import 'package:easy_localization/easy_localization.dart';
import 'package:flutter/material.dart';
import 'package:settings_ui/settings_ui.dart';

class Proxy extends StatelessWidget {
  const Proxy({
    Key? key,
  }) : super(key: key);

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: const Text('Settings.Proxy.Label').tr(),
      ),
      body: SettingsList(sections: [
        SettingsSection(
          title: const Text('Settings.Proxy.Enabled').tr(),
          tiles: [
            SettingsTile(
              title: const Text('Settings.Proxy.Enabled').tr(),
              value: ListView.separated(
                shrinkWrap: true,
                  itemBuilder: (context, index) {
                    return ListTile(
                      title: Text('item $index'),
                    );
                  },
                  separatorBuilder: (context, index) => const Divider(),
                  itemCount: 5),
            ),
            SettingsTile.switchTile(
              onToggle: (value) {},
              initialValue: true,
              title: const Text('Settings.Notifications.Enabled').tr(),
              description: const Text('Settings.Proxy.Footer').tr(),
            ),
          ],
        )
      ]),
    );
  }
}
