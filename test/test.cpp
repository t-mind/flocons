#include "test.h"

int main(int argc, char **argv) {
    //--- Create the event manager and test controller
    CPPUNIT_NS::TestResult controller;

    CPPUNIT_NS::TestResultCollector result_collector;
    controller.addListener(&result_collector);

    //--- Add a listener that colllects test result
    CPPUNIT_NS::TextTestResult /*TestResultCollector*/ result;
    controller.addListener(&result);

    //--- Add a listener that print dots as test run.
    CPPUNIT_NS::BriefTestProgressListener progress;
    controller.addListener(&progress);

    //--- Add the top suite to the test runner
    CPPUNIT_NS::TestRunner runner;
    if (argc == 1)
        runner.addTest(
            CPPUNIT_NS::TestFactoryRegistry::getRegistry().makeTest());
    else
        for (int i = 1; i < argc; ++i) {
            runner.addTest(CPPUNIT_NS::TestFactoryRegistry::getRegistry(argv[i])
                               .makeTest());
        }
    runner.run(controller);

    CPPUNIT_NS::CompilerOutputter outputter(&result, CPPUNIT_NS::stdCOut());
    outputter.write();

    // Uncomment this for XML output
    std::ofstream file("cppunit-report.xml");
    CPPUNIT_NS::XmlOutputter xml(&result_collector, file);
    xml.write();
    file.close();

    if (result.wasSuccessful()) {
        return 0;
    } else {
        result.print(std::cerr);
        return 1;
    }
}