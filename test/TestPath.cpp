#include "../src/file/Path.h"
#include "test.h"

class TestPath : public CPPUNIT_NS::TestFixture {
    CPPUNIT_TEST_SUITE(TestPath);
    CPPUNIT_TEST(testSimplePath);
    CPPUNIT_TEST_SUITE_END();

  public:
    void setUp(void) {}
    void tearDown(void) {}

  protected:
    void testSimplePath(void);
};

void TestPath::testSimplePath() {
    Path p("/test");
    CPPUNIT_ASSERT_EQUAL(Path("/test"), p);

    p = "";
    CPPUNIT_ASSERT_EQUAL(Path("/"), p);

    Path p2 = p / "test";
    CPPUNIT_ASSERT_EQUAL(Path("/test"), p2);

    Path p3("myName");
    CPPUNIT_ASSERT_EQUAL(std::string("myName"), p3.basename());
}

CPPUNIT_TEST_SUITE_REGISTRATION(TestPath);
CPPUNIT_TEST_SUITE_NAMED_REGISTRATION(TestPath, "TestPath");