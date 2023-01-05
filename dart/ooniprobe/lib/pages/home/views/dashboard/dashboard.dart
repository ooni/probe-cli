import 'package:easy_localization/easy_localization.dart';
import 'package:flutter/material.dart';
import 'package:flutter_svg/svg.dart';
import 'package:shape_of_view_null_safe/shape_of_view_null_safe.dart';
import 'package:shared/pages/home/views/dashboard/test_overview.dart';
import 'package:shared/pages/home/views/dashboard/tests/tests.dart';

class Dashboard extends StatelessWidget {
  const Dashboard({Key? key}) : super(key: key);

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      body: SingleChildScrollView(
        child: Column(
          children: <Widget>[
            Container(
              width: double.infinity,
              color: Theme.of(context).primaryColor,
              child: Stack(
                children: [
                  Column(
                    mainAxisAlignment: MainAxisAlignment.start,
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      const SizedBox(height: 20),
                      Padding(
                        padding: const EdgeInsets.symmetric(horizontal: 16),
                        child: SizedBox(
                          height: 80,
                          width: 200,
                          child: SvgPicture.network(
                            'https://raw.githubusercontent.com/ooni/design-system/master/components/svgs/logos/Probe-HorizontalMonochromeInverted.svg',
                            semanticsLabel: 'OONI-HorizontalMonochromeInverted',
                            placeholderBuilder: (BuildContext context) =>
                                Container(),
                          ),
                        ),
                      ),
                      const SizedBox(height: 40),
                      SizedBox(
                        width: double.infinity,
                        child: ShapeOfView(
                          elevation: 0,
                          shape: ArcShape(
                            direction: ArcDirection.Inside,
                            height: 30,
                            position: ArcPosition.Bottom,
                          ),
                        ),
                      ),
                      Container(
                        padding: const EdgeInsets.only(top: 10),
                        color: Colors.white,
                        child: Row(
                          mainAxisAlignment: MainAxisAlignment.center,
                          children: [
                            const Text('Dashboard.Overview.LatestTest').tr(),
                          ],
                        ),
                      )
                    ],
                  ),
                  Positioned.fill(
                    top: 60,
                    child: Align(
                      alignment: Alignment.center,
                      child: ElevatedButton(
                        style: ButtonStyle(
                          backgroundColor:
                              MaterialStateProperty.all(Colors.white),
                        ),
                        onPressed: () {},
                        child: Padding(
                          padding: const EdgeInsets.all(6.0),
                          child: Text(
                            'Dashboard.Card.Run',
                            style: TextStyle(
                              color: Theme.of(context).primaryColor,
                              fontSize: 25,
                            ),
                          ).tr(),
                        ),
                      ),
                    ),
                  ),
                ],
              ),
            ),
            ListView.builder(
              itemCount: tests.length,
              shrinkWrap: true,
              padding: EdgeInsets.zero,
              physics: const NeverScrollableScrollPhysics(),
              itemBuilder: (BuildContext context, int index) {
                return Padding(
                  padding: const EdgeInsets.all(8.0),
                  child: InkWell(
                    onTap: () {
                      Navigator.of(context).push(
                        TestOverViewPage.route(descriptor: tests[index]),
                      );
                    },
                    child: Card(
                        child: Padding(
                          padding: const EdgeInsets.all(4.0),
                          child: Row(
                      mainAxisAlignment: MainAxisAlignment.start,
                      mainAxisSize: MainAxisSize.min,
                      crossAxisAlignment: CrossAxisAlignment.start,
                      children: [
                          Padding(
                            padding: const EdgeInsets.all(8.0),
                            child: Icon(
                              tests[index].icon,
                              size: 60,
                              color: Colors.grey.shade600,
                            ),
                          ),
                          Flexible(
                            child: Column(
                              crossAxisAlignment: CrossAxisAlignment.start,
                              mainAxisSize: MainAxisSize.min,
                              mainAxisAlignment: MainAxisAlignment.start,
                              children: [
                                Padding(
                                  padding:
                                      const EdgeInsets.symmetric(vertical: 10.0),
                                  child: Text(
                                    tests[index].name,
                                    style: Theme.of(context).textTheme.headline6,
                                  ),
                                ),
                                Text(tests[index].description),
                              ],
                            ),
                          ),
                      ],
                    ),
                        )

                        // ListTile(
                        //   leading: ,
                        //   title: Text(tests[index].name),
                        //   subtitle: Text(tests[index].description),
                        // ),
                        ),
                  ),
                );
              },
            ),
          ],
        ),
      ),
    );
  }
}
