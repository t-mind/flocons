#include "../src/local/LocalContext.h"
#include "../src/local/LocalFileService.h"
#include "test.h"

class TestDirectory : public CPPUNIT_NS::TestFixture {
    CPPUNIT_TEST_SUITE(TestDirectory);
    CPPUNIT_TEST(testLS);
    CPPUNIT_TEST_SUITE_END();

  private:
    std::string _mount_point;
    std::unique_ptr<FileService> _file_service;

  public:
    void setUp(void) {
        char tmp_template[] = "/tmp/fstest.XXXXXX";
        this->_mount_point = mkdtemp(tmp_template);
        this->_file_service = std::make_unique<LocalFileService>("test", this->_mount_point);
    }
    void tearDown(void) { system(("rm -rf " + this->_mount_point).c_str()); }

  protected:
    void testLS(void);
};

void TestDirectory::testLS() {
    bool error_catched = false;
    try {
        this->_file_service->getFile("/lol");
    } catch (std::system_error e) {
        error_catched = true;
        CPPUNIT_ASSERT_EQUAL(ENOENT, e.code().value());
    }
    CPPUNIT_ASSERT_EQUAL(true, error_catched);

    auto f2 = this->_file_service->createDirectory("/lol");
    CPPUNIT_ASSERT(f2 != nullptr);
    CPPUNIT_ASSERT_EQUAL((mode_t)0755, f2->mode());

    auto f3 = this->_file_service->getFile("/lol");
    CPPUNIT_ASSERT(f2 == f3);
}

CPPUNIT_TEST_SUITE_REGISTRATION(TestDirectory);
CPPUNIT_TEST_SUITE_NAMED_REGISTRATION(TestDirectory, "TestDirectory");