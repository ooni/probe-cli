import 'package:flutter/material.dart';
import 'package:material_design_icons_flutter/material_design_icons_flutter.dart';
import 'package:shared/pages/home/home.dart';

class DashboardPage extends StatefulWidget {
  const DashboardPage({Key? key}) : super(key: key);

  @override
  State<DashboardPage> createState() => _DashboardPageState();
}

class _DashboardPageState extends State<DashboardPage> {
  final _pageViewController = PageController();

  int _activePage = 0;

  @override
  void dispose() {
    _pageViewController.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      body: PageView(
        controller: _pageViewController,
        children: const <Widget>[
          Dashboard(),
          TestResult(),
          Settings()
        ],
        onPageChanged: (index) {
          setState(() {
            _activePage = index;
          });
        },
      ),
      bottomNavigationBar: Column(
        mainAxisAlignment: MainAxisAlignment.end,
        mainAxisSize: MainAxisSize.min,
        children: [
          const TestProgressIndicator(),
          BottomNavigationBar(
            currentIndex: _activePage,
            onTap: (index) {
              _pageViewController.animateToPage(index,
                  duration: const Duration(milliseconds: 200),
                  curve: Curves.bounceOut);
            },
            items: const [
              BottomNavigationBarItem(
                icon: Icon(Icons.web_sharp),
                label: "Dashboard",
              ),
              BottomNavigationBarItem(
                icon: Icon(MdiIcons.history),
                label: "Test Results",
              ),
              BottomNavigationBarItem(
                icon: Icon(MdiIcons.cogOutline),
                label: "Settings",
              ),
            ],
          ),
        ],
      ),
    );
  }
}
