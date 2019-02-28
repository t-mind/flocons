#ifndef TEST_H_
#define TEST_H_

#include <cppunit/BriefTestProgressListener.h>
#include <cppunit/CompilerOutputter.h>
#include <cppunit/TestResult.h>
#include <cppunit/TestResultCollector.h>
#include <cppunit/TestRunner.h>
#include <cppunit/TextTestResult.h>
#include <cppunit/XmlOutputter.h>
#include <cppunit/extensions/HelperMacros.h>
#include <cppunit/extensions/TestFactoryRegistry.h>

#include "../src/common.h"
#include "../src/file/Path.h"
#include "../src/http/URL.h"

using namespace flocons;

#define FAIL(msg)                                                                                                                                              \
    {                                                                                                                                                          \
        std::stringstream str;                                                                                                                                 \
        str << "Error: " << msg << std::endl;                                                                                                                  \
        CPPUNIT_NS::Asserter::fail(str.str(), CPPUNIT_SOURCELINE());                                                                                           \
    }

namespace CppUnit {
template <> struct assertion_traits<Path> {
    static bool equal(const Path& a, const Path& b) { return a == b; }
    static std::string toString(const Path& p) { return p.string(); }
};
template <> struct assertion_traits<URL> {
    static bool equal(const URL& a, const URL& b) { return a == b; }
    static std::string toString(const URL& p) { return p.string(); }
};
} // namespace CppUnit

#endif /* TEST_H_ */