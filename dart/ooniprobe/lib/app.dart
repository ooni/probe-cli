import 'dart:async';
import 'dart:io';

import 'package:flutter/cupertino.dart' hide MenuItem;
import 'package:flutter/material.dart' hide MenuItem;
import 'package:easy_localization/easy_localization.dart';
import 'package:flutter_bloc/flutter_bloc.dart';
import 'package:shared/pages/about/about.dart';
import 'package:shared/pages/home/bloc/run_test_bloc.dart';
import 'package:shared/pages/home/home.dart';
import 'package:tray_manager/tray_manager.dart';

part 'theme_utils.dart';

const _kIconTypeDefault = 'default';
const _kIconTypeOriginal = 'original';

class App extends StatefulWidget {
  const App({Key? key}) : super(key: key);

  @override
  State<App> createState() => _AppState();
}

class _AppState extends State<App> with TrayListener {
  @override
  void initState() {
    super.initState();
    initTrayIcon();
  }

  @override
  void dispose() {
    trayManager.removeListener(this);
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    return BlocProvider(
      create: (context) => RunTestBloc(),
      child: MaterialApp(
        title: 'OONI Probe',
        localizationsDelegates: context.localizationDelegates,
        supportedLocales: context.supportedLocales,
        locale: context.locale,
        theme: _themeData(ThemeMode.light, context),
        darkTheme: _themeData(ThemeMode.dark, context),
        initialRoute: '/dashboard',
        routes: {
          '/about': (context) => const AboutPage(),
          '/dashboard': (context) => const DashboardPage(),
        },
      ),
    );
  }

  @override
  void onTrayIconMouseDown() {
    print('onTrayIconMouseDown');
    trayManager.popUpContextMenu();
  }

  @override
  void onTrayIconMouseUp() {
    print('onTrayIconMouseUp');
  }

  @override
  void onTrayIconRightMouseDown() {
    print('onTrayIconRightMouseDown');
    // trayManager.popUpContextMenu();
  }

  @override
  void onTrayIconRightMouseUp() {
    print('onTrayIconRightMouseUp');
  }

  @override
  void onTrayMenuItemClick(MenuItem menuItem) {
    print(menuItem.toJson());
    print('${menuItem.toJson()}');
  }

  void initTrayIcon() async {
    trayManager.addListener(this);
    await trayManager.setIcon('assets/images/logo.png');
    await trayManager.setToolTip('tray_manager');
    await trayManager.setTitle('tray_manager');
    Menu _menu = Menu(
      items: [
        MenuItem(
          label: 'Look Up "OONI Probe"',
        ),
        MenuItem(
          label: 'Run All Tesys',
        ),
        MenuItem.separator(),
        MenuItem(
          label: 'Cut',
        ),
        MenuItem(
          label: 'Copy',
        ),
        MenuItem(
          label: 'Paste',
          disabled: true,
        ),
        MenuItem.submenu(
          label: 'Share',
          submenu: Menu(
            items: [
              MenuItem.checkbox(
                label: 'Item 1',
                checked: true,
                onClick: (menuItem) {
                  print('click item 1');
                  menuItem.checked = !(menuItem.checked == true);
                },
              ),
              MenuItem.checkbox(
                label: 'Item 2',
                checked: false,
                onClick: (menuItem) {
                  print('click item 2');
                  menuItem.checked = !(menuItem.checked == true);
                },
              ),
            ],
          ),
        ),
        MenuItem.separator(),
        MenuItem.submenu(
          label: 'Font',
          submenu: Menu(
            items: [
              MenuItem.checkbox(
                label: 'Item 1',
                checked: true,
                onClick: (menuItem) {
                  print('click item 1');
                  menuItem.checked = !(menuItem.checked == true);
                },
              ),
              MenuItem.checkbox(
                label: 'Item 2',
                checked: false,
                onClick: (menuItem) {
                  print('click item 2');
                  menuItem.checked = !(menuItem.checked == true);
                },
              ),
              MenuItem.separator(),
              MenuItem(
                label: 'Item 3',
                checked: false,
              ),
              MenuItem(
                label: 'Item 4',
                checked: false,
              ),
              MenuItem(
                label: 'Item 5',
                checked: false,
              ),
            ],
          ),
        ),
        MenuItem.submenu(
          label: 'Speech',
          submenu: Menu(
            items: [
              MenuItem(
                label: 'Item 1',
              ),
              MenuItem(
                label: 'Item 2',
              ),
            ],
          ),
        ),
      ],
    );
    await trayManager.setContextMenu(_menu);
  }
}
