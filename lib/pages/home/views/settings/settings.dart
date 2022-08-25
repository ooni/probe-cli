import 'package:easy_localization/easy_localization.dart';
import 'package:flutter/material.dart';
import 'package:shared/pages/home/views/settings/advanced.dart';
import 'package:shared/pages/home/views/settings/automated_testing.dart';
import 'package:shared/pages/home/views/settings/notifications.dart';
import 'package:shared/pages/home/views/settings/privacy.dart';
import 'package:shared/pages/home/views/settings/proxy.dart';
import 'package:shared/utils/navigation.dart';
import 'package:settings_ui/settings_ui.dart';
import 'package:url_launcher/url_launcher.dart';

String? encodeQueryParameters(Map<String, String> params) {
  return params.entries
      .map((MapEntry<String, String> e) =>
  '${Uri.encodeComponent(e.key)}=${Uri.encodeComponent(e.value)}')
      .join('&');
}

class Settings extends StatelessWidget {
  const Settings({
    Key? key,
  }) : super(key: key);

  @override
  Widget build(BuildContext context) {

    return Scaffold(
      appBar: AppBar(
        title: const Text('Settings.Title').tr(),
        centerTitle: false,
        elevation: 0,
      ),
      body: SettingsList(
        platform: DevicePlatform.android,
        sections: [
          SettingsSection(
            tiles: <SettingsTile>[
              SettingsTile(
                title: const Text('Settings.Notifications.Label').tr(),
                leading: const Icon(Icons.notifications),
                trailing: const Icon(Icons.keyboard_arrow_right_outlined),
                onPressed: (context) {
                  Navigation.navigateTo(
                    context: context,
                    screen: const Notifications(),
                  );
                },
              ),
              SettingsTile(
                title: const Text('Settings.AutomatedTesting.Label').tr(),
                leading: const Icon(Icons.change_circle_outlined),
                trailing: const Icon(Icons.keyboard_arrow_right_outlined),
                onPressed: (context) {
                  Navigation.navigateTo(
                    context: context,
                    screen: const AutomatedTesting(),
                  );
                },
              ),
              SettingsTile.navigation(
                leading: const Icon(Icons.settings),
                trailing: const Icon(Icons.keyboard_arrow_right_outlined),
                title: const Text('Settings.TestOptions.Label').tr(),
              ),
              SettingsTile.navigation(
                leading: const Icon(Icons.lock),
                trailing: const Icon(Icons.keyboard_arrow_right_outlined),
                title: const Text('Settings.Privacy.Label').tr(),
                onPressed: (context) {
                  Navigation.navigateTo(
                    context: context,
                    screen: const Privacy(),
                  );
                },
              ),
              SettingsTile.navigation(
                leading: const Icon(Icons.more_horiz),
                trailing: const Icon(Icons.keyboard_arrow_right_outlined),
                title: const Text('Settings.Advanced.Label').tr(),
                onPressed: (context) {
                  Navigation.navigateTo(
                    context: context,
                    screen: const  Advanced(),
                  );
                },
              ),
              SettingsTile.navigation(
                leading: const Icon(Icons.switch_right),
                trailing: const Icon(Icons.keyboard_arrow_right_outlined),
                title: const Text('Settings.Proxy.Label').tr(),
                onPressed: (context) {
                  Navigation.navigateTo(
                    context: context,
                    screen: const Proxy(),
                  );
                },
              ),
              SettingsTile.navigation(
                leading: const Icon(Icons.question_mark_rounded),
                trailing: const Icon(Icons.keyboard_arrow_right_outlined),
                title: const Text('Settings.SendEmail.Label').tr(),
                onPressed: (context)  async {
                  var uri = Uri(
                    scheme: 'mailto',
                    path: 'contact@openobservatory.org',
                    query: encodeQueryParameters(<String, String>{
                      'subject': '[bug-report] OONI Probe Android %1',
                      'body':'${'Settings.SendEmail.Message'.tr()} \n MANUFACTURER:  \nMODEL:   \nBOARD:  \nTIME: '
                    }),
                  );
                  if (!await launchUrl(uri)) {
                    ScaffoldMessenger.of(context).showSnackBar(
                      SnackBar(
                        content: Text('Could not launch ${uri.toString()}'),
                      ),
                    );
                  }
                },
              ),
              SettingsTile.navigation(
                leading: const Icon(Icons.circle_notifications_outlined),
                trailing: const Icon(Icons.keyboard_arrow_right_outlined),
                title: const Text('Settings.About.Label').tr(),
                onPressed: (context) {
                  Navigator.of(context).pushNamed('/about');
                },
              ),
            ],
          ),
        ],
      )
    );
  }
}
