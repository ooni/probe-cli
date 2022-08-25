
import 'package:flutter/material.dart';
import 'package:easy_localization/easy_localization.dart';
import 'package:flutter_bloc/flutter_bloc.dart';
import 'package:shared/pages/about/about.dart';
import 'package:shared/pages/home/bloc/run_test_bloc.dart';
import 'package:shared/pages/home/home.dart';


part 'theme_utils.dart';

class App extends StatelessWidget {
  const App({Key? key}) : super(key: key);

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
}
