#include "../src/local/LocalContext.h"
#include "../src/local/LocalFileService.h"
#include "test.h"

class TestConcurrentService : public CPPUNIT_NS::TestFixture {
    CPPUNIT_TEST_SUITE(TestConcurrentService);
    CPPUNIT_TEST(testLS);
    CPPUNIT_TEST_SUITE_END();

  private:
    std::string _mount_point;
    std::unique_ptr<FileService> _file_service_1;
    std::unique_ptr<FileService> _file_service_2;

  public:
    void setUp(void) {
        char tmp_template[] = "/tmp/fstest.XXXXXX";
        this->_mount_point = mkdtemp(tmp_template);
        this->_file_service_1 = std::make_unique<LocalFileService>("test1", this->_mount_point);
        this->_file_service_2 = std::make_unique<LocalFileService>("test2", this->_mount_point);
    }
    void tearDown(void) {
        // system(("rm -rf " + this->_mount_point).c_str());
    }

  protected:
    void testLS(void);
};

void TestConcurrentService::testLS() {
    bool error_catched = false;
    try {
        this->_file_service_1->getFile("/lol");
    } catch (std::system_error e) {
        error_catched = true;
        CPPUNIT_ASSERT_EQUAL(ENOENT, e.code().value());
    }
    CPPUNIT_ASSERT_EQUAL(true, error_catched);

    auto f2 = this->_file_service_2->createDirectory("/lol");
    CPPUNIT_ASSERT(f2 != nullptr);
    CPPUNIT_ASSERT_EQUAL((mode_t)0755, f2->mode());

    auto f3 = this->_file_service_1->getFile("/lol");
    CPPUNIT_ASSERT(f3 != nullptr);

    const char* content = "my test content";
    auto f4 = this->_file_service_1->createRegularFile("/lol/testFyle", content, strlen(content) + 1);
    CPPUNIT_ASSERT(f4 != nullptr);

    auto f5 = this->_file_service_2->getRegularFile("/lol/testFyle");
    CPPUNIT_ASSERT(f5 != nullptr);
    CPPUNIT_ASSERT(strcmp(content, f5->data()) == 0);

    const char* content2 = "my test content 2";
    auto f6 = this->_file_service_1->createRegularFile("/lol/testFyle2", content2, strlen(content2) + 1);
    CPPUNIT_ASSERT(f6 != nullptr);

    auto f7 = this->_file_service_2->getRegularFile("/lol/testFyle2");
    CPPUNIT_ASSERT(f7 != nullptr);
    CPPUNIT_ASSERT(strcmp(content2, f7->data()) == 0);
}

CPPUNIT_TEST_SUITE_REGISTRATION(TestConcurrentService);
CPPUNIT_TEST_SUITE_NAMED_REGISTRATION(TestConcurrentService, "TestConcurrentService");