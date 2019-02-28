#include "../src/http/URL.h"
#include "test.h"

class TestURL : public CPPUNIT_NS::TestFixture {
    CPPUNIT_TEST_SUITE(TestURL);
    CPPUNIT_TEST(testSimpleURL);
    CPPUNIT_TEST_SUITE_END();

  public:
    void setUp(void) {}
    void tearDown(void) {}

  protected:
    void testSimpleURL(void);
};

void TestURL::testSimpleURL() {
    URL p("http://localhost");
    CPPUNIT_ASSERT_EQUAL(URL("http://localhost"), p);
    CPPUNIT_ASSERT_EQUAL(std::string("http"), p.protocol());
    CPPUNIT_ASSERT_EQUAL(std::string("localhost"), p.hostname());
    CPPUNIT_ASSERT_EQUAL(80, p.port());

    p = "https://localhost";
    CPPUNIT_ASSERT_EQUAL(URL("https://localhost"), p);
    CPPUNIT_ASSERT_EQUAL(std::string("https"), p.protocol());
    CPPUNIT_ASSERT_EQUAL(std::string("localhost"), p.hostname());
    CPPUNIT_ASSERT_EQUAL(443, p.port());

    URL p2 = p / "test";
    CPPUNIT_ASSERT_EQUAL(URL("https://localhost/test"), p2);
    CPPUNIT_ASSERT_EQUAL(std::string("https://localhost/test"), p2.string());

    p2 = p / "/test";
    CPPUNIT_ASSERT_EQUAL(URL("https://localhost/test"), p2);
    CPPUNIT_ASSERT_EQUAL(std::string("https://localhost/test"), p2.string());

    p = "https://test:8000";
    CPPUNIT_ASSERT_EQUAL(std::string("https"), p.protocol());
    CPPUNIT_ASSERT_EQUAL(std::string("test"), p.hostname());
    CPPUNIT_ASSERT_EQUAL(8000, p.port());

    p2 = p / "test";
    CPPUNIT_ASSERT_EQUAL(URL("https://test:8000/test"), p2);
    CPPUNIT_ASSERT_EQUAL(std::string("https://test:8000/test"), p2.string());

    p2 = p / "/test";
    CPPUNIT_ASSERT_EQUAL(URL("https://test:8000/test"), p2);
    CPPUNIT_ASSERT_EQUAL(std::string("https://test:8000/test"), p2.string());
}

CPPUNIT_TEST_SUITE_REGISTRATION(TestURL);
CPPUNIT_TEST_SUITE_NAMED_REGISTRATION(TestURL, "TestURL");