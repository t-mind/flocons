#include "../src/local/HTTPFileServer.h"
#include "../src/local/LocalContext.h"
#include "../src/local/LocalFileService.h"
#include "../src/remote/HTTPFileService.h"
#include "test.h"
#include <curl/curl.h>

class TestHTTPFileServerAndService : public CPPUNIT_NS::TestFixture {
    CPPUNIT_TEST_SUITE(TestHTTPFileServerAndService);
    CPPUNIT_TEST(testGet);
    CPPUNIT_TEST_SUITE_END();

  private:
    std::string _mount_point;
    std::shared_ptr<FileService> _local_file_service;
    std::shared_ptr<FileService> _remote_file_service;
    std::unique_ptr<HTTPFileServer> _http_server;
    int _port = 10789;

  public:
    void setUp(void) {
        char tmp_template[] = "/tmp/fstest.XXXXXX";
        this->_mount_point = mkdtemp(tmp_template);
        this->_local_file_service = std::make_shared<LocalFileService>("test", this->_mount_point);
        this->_remote_file_service = std::make_shared<HTTPFileService>("http://127.0.0.1:" + std::to_string(this->_port));
        this->_http_server = std::make_unique<HTTPFileServer>(this->_local_file_service, this->_port);
        this->_http_server->start();
    }
    void tearDown(void) {
        system(("rm -rf " + this->_mount_point).c_str());
        this->_http_server->stop();
    }

  protected:
    void testGet(void);
};

void TestHTTPFileServerAndService::testGet() {
    this->_local_file_service->createDirectory("/test");
    auto directory = this->_remote_file_service->getDirectory("/test");

    CPPUNIT_ASSERT_EQUAL(Path("/test"), directory->path());

    this->_remote_file_service->createDirectory("/test2");
    directory = this->_local_file_service->getDirectory("/test2");

    CPPUNIT_ASSERT_EQUAL(Path("/test2"), directory->path());

    const char* data = "my test data";
    this->_local_file_service->createRegularFile("/test/myFile", (char*)data, strlen(data) + 1);
    auto file = this->_remote_file_service->getRegularFile("/test/myFile");

    CPPUNIT_ASSERT_EQUAL(std::string(data), std::string(file->data()));

    this->_remote_file_service->createRegularFile("/test2/myFile2", (char*)data, strlen(data) + 1);
    file = this->_local_file_service->getRegularFile("/test2/myFile2");

    CPPUNIT_ASSERT_EQUAL(std::string(data), std::string(file->data()));
}

CPPUNIT_TEST_SUITE_REGISTRATION(TestHTTPFileServerAndService);
CPPUNIT_TEST_SUITE_NAMED_REGISTRATION(TestHTTPFileServerAndService, "TestHTTPFileServerAndService");