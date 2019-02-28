#include "../src/local/HTTPFileServer.h"
#include "../src/local/LocalContext.h"
#include "../src/local/LocalFileService.h"
#include "test.h"
#include <curl/curl.h>
#include <netinet/in.h>
#include <unistd.h>

class TestHTTPFileServer : public CPPUNIT_NS::TestFixture {
    CPPUNIT_TEST_SUITE(TestHTTPFileServer);
    CPPUNIT_TEST(testConcurrent);
    CPPUNIT_TEST(testGet);
    CPPUNIT_TEST_SUITE_END();

  private:
    std::string _mount_point;
    std::shared_ptr<FileService> _file_service;
    std::unique_ptr<HTTPFileServer> _http_server;
    int _port = 10789;

  public:
    void setUp(void) {
        char tmp_template[] = "/tmp/fstest.XXXXXX";
        this->_mount_point = mkdtemp(tmp_template);
        this->_file_service = std::make_shared<LocalFileService>("test", this->_mount_point);
        this->_http_server = std::make_unique<HTTPFileServer>(this->_file_service, this->_port);
        this->_http_server->start();
    }
    void tearDown(void) {
        system(("rm -rf " + this->_mount_point).c_str());
        this->_http_server->stop();
    }

  protected:
    void testConcurrent(void);
    void testGet(void);
};

size_t writeFunction(void* ptr, size_t size, size_t nmemb, std::string* data) {
    data->append((char*)ptr, size * nmemb);
    return size * nmemb;
}

void TestHTTPFileServer::testConcurrent() {
    int server_fd = socket(AF_INET, SOCK_STREAM, 0);
    struct sockaddr_in server_config;
    server_config.sin_family = AF_INET;
    server_config.sin_port = htons(this->_port + 1);
    server_config.sin_addr.s_addr = htonl(INADDR_ANY);
    bind(server_fd, (struct sockaddr*)&server_config, sizeof(server_config));
    listen(server_fd, 128);

    HTTPFileServer server(this->_file_service, this->_port + 1);
    int catched_error = 0;
    try {
        server.start();
    } catch (std::system_error e) {
        catched_error = e.code().value();
    }
    CPPUNIT_ASSERT_EQUAL(EADDRINUSE, catched_error);
    server.stop();
    close(server_fd);
    shutdown(server_fd, SHUT_RDWR);
}

void TestHTTPFileServer::testGet() {
    this->_file_service->createDirectory("/test");
    const char* data = "my test data";

    std::string base_url = "http://127.0.0.1:" + std::to_string(this->_port);
    CURL* curl = curl_easy_init();
    if (curl) {
        std::string response;
        char errorBuffer[CURL_ERROR_SIZE];
        // HEAD
        curl_easy_setopt(curl, CURLOPT_URL, (base_url + "/test/myFile").c_str());
        curl_easy_setopt(curl, CURLOPT_ERRORBUFFER, errorBuffer);
        curl_easy_setopt(curl, CURLOPT_FOLLOWLOCATION, 1L);
        curl_easy_setopt(curl, CURLOPT_WRITEFUNCTION, writeFunction);
        curl_easy_setopt(curl, CURLOPT_WRITEDATA, &response);
        curl_easy_setopt(curl, CURLOPT_NOBODY, 1L); // Make head query
        auto res = curl_easy_perform(curl);
        CPPUNIT_ASSERT_EQUAL(std::string(), std::string(errorBuffer));
        CPPUNIT_ASSERT(res == CURLE_OK);
        long http_code;
        curl_easy_getinfo(curl, CURLINFO_RESPONSE_CODE, &http_code);
        CPPUNIT_ASSERT_EQUAL((long)404, http_code);

        // PUT
        curl_easy_reset(curl);
        curl_easy_setopt(curl, CURLOPT_URL, (base_url + "/test/myFile").c_str());
        curl_easy_setopt(curl, CURLOPT_ERRORBUFFER, errorBuffer);
        curl_easy_setopt(curl, CURLOPT_FOLLOWLOCATION, 1L);
        curl_easy_setopt(curl, CURLOPT_WRITEFUNCTION, writeFunction);
        curl_easy_setopt(curl, CURLOPT_WRITEDATA, &response);

        curl_easy_setopt(curl, CURLOPT_CUSTOMREQUEST, "PUT");
        curl_easy_setopt(curl, CURLOPT_POSTFIELDS, data);
        curl_easy_setopt(curl, CURLOPT_POSTFIELDSIZE, strlen(data) + 1);
        res = curl_easy_perform(curl);
        CPPUNIT_ASSERT_EQUAL(std::string(), std::string(errorBuffer));
        CPPUNIT_ASSERT(res == CURLE_OK);

        curl_easy_getinfo(curl, CURLINFO_RESPONSE_CODE, &http_code);
        CPPUNIT_ASSERT_EQUAL((long)200, http_code);

        // HEAD
        curl_easy_reset(curl);
        curl_easy_setopt(curl, CURLOPT_URL, (base_url + "/test/myFile").c_str());
        curl_easy_setopt(curl, CURLOPT_ERRORBUFFER, errorBuffer);
        curl_easy_setopt(curl, CURLOPT_FOLLOWLOCATION, 1L);
        curl_easy_setopt(curl, CURLOPT_WRITEFUNCTION, writeFunction);
        curl_easy_setopt(curl, CURLOPT_WRITEDATA, &response);
        curl_easy_setopt(curl, CURLOPT_NOBODY, 1L); // Make head query
        res = curl_easy_perform(curl);
        CPPUNIT_ASSERT_EQUAL(std::string(), std::string(errorBuffer));
        CPPUNIT_ASSERT(res == CURLE_OK);

        curl_easy_getinfo(curl, CURLINFO_RESPONSE_CODE, &http_code);
        CPPUNIT_ASSERT(http_code == 200);
        curl_off_t content_length;
        curl_easy_getinfo(curl, CURLINFO_CONTENT_LENGTH_DOWNLOAD_T, &content_length);
        CPPUNIT_ASSERT_EQUAL(strlen(data) + 1, (size_t)content_length);

        // GET
        curl_easy_reset(curl);
        curl_easy_setopt(curl, CURLOPT_URL, (base_url + "/test/myFile").c_str());
        curl_easy_setopt(curl, CURLOPT_ERRORBUFFER, errorBuffer);
        curl_easy_setopt(curl, CURLOPT_FOLLOWLOCATION, 1L);
        curl_easy_setopt(curl, CURLOPT_WRITEFUNCTION, writeFunction);
        curl_easy_setopt(curl, CURLOPT_WRITEDATA, &response);

        res = curl_easy_perform(curl);
        CPPUNIT_ASSERT(res == CURLE_OK);
        curl_easy_getinfo(curl, CURLINFO_RESPONSE_CODE, &http_code);
        CPPUNIT_ASSERT_EQUAL((long)200, http_code);
        curl_easy_getinfo(curl, CURLINFO_CONTENT_LENGTH_DOWNLOAD_T, &content_length);
        CPPUNIT_ASSERT_EQUAL(strlen(data) + 1, (size_t)content_length);
        // Let's compare data with reponse where we remove last character (we receive one \0 char from connection)
        CPPUNIT_ASSERT_EQUAL(std::string(data), response.substr(0, response.length() - 1));

        curl_easy_cleanup(curl);
    }
}

CPPUNIT_TEST_SUITE_REGISTRATION(TestHTTPFileServer);
CPPUNIT_TEST_SUITE_NAMED_REGISTRATION(TestHTTPFileServer, "TestHTTPFileServer");