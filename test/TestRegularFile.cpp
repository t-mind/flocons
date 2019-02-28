#include "../src/local/LocalContext.h"
#include "../src/local/LocalFileService.h"
#include "test.h"

class TestRegularFile : public CPPUNIT_NS::TestFixture {
    CPPUNIT_TEST_SUITE(TestRegularFile);
    CPPUNIT_TEST(testWrite);
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
    void testWrite(void);
};

void TestRegularFile::testWrite() {
    this->_file_service->createDirectory("/test");
    const char* data = "my test data";
    this->_file_service->createRegularFile("/test/myFile", (char*)data, strlen(data) + 1);

    auto file = this->_file_service->getRegularFile("/test/myFile");
    CPPUNIT_ASSERT(file != nullptr);
    CPPUNIT_ASSERT(strcmp(data, file->data()) == 0);

    this->_file_service->createRegularFile("/test/myFile2", (char*)data, strlen(data) + 1);

    file = this->_file_service->getRegularFile("/test/myFile2");
    CPPUNIT_ASSERT(file != nullptr);
    CPPUNIT_ASSERT(strcmp(data, file->data()) == 0);
}

CPPUNIT_TEST_SUITE_REGISTRATION(TestRegularFile);
CPPUNIT_TEST_SUITE_NAMED_REGISTRATION(TestRegularFile, "TestRegularFile");