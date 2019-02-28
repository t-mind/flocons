#ifndef _FLOCONS_HTTP_REQUEST_H_
#define _FLOCONS_HTTP_REQUEST_H_

#include "../common.h"
#include "URL.h"
#include "http.h"

namespace flocons {

class HTTPRequest {
  private:
    void* _handle = NULL;
    bool _locked = false;
    std::mutex _mutex;
    URL _base_url;

    char _error_buffer[256];
    char* _response_buffer = NULL;
    size_t _response_buffer_size = 0;
    size_t _response_buffer_max_size = 0;
    std::string _location;
    std::string _mime_type;
    mode_t _mode;
    time_t _modification_time;

    void _reset();
    static size_t _data_cb(char* ptr, size_t size, size_t nb, void* data);
    static size_t _header_cb(char* ptr, size_t size, size_t nb, void* data);

  public:
    HTTPRequest(const URL& base_url) : _base_url(base_url) {}
    ~HTTPRequest();

    const URL& baseUrl() const { return this->_base_url; }

    bool try_lock();
    void unlock();

    HTTPCode head(const std::string& relative_url);
    HTTPCode get(const std::string& relative_url);
    HTTPCode put(const std::string& relative_url, const char* data, size_t size, const std::string& mime_type = "");

    const char* data() const { return this->_response_buffer; }
    size_t dataSize() const { return this->_response_buffer_size; }
    const std::string& mimeType() const { return this->_mime_type; }
    mode_t mode() const { return this->_mode; }
    time_t modification_time() const { return this->_modification_time; }
    const std::string& location() const { return this->_location; }
    std::mutex& mutex() { return this->_mutex; }
};

} // namespace flocons

#endif // !_FLOCONS_HTTP_REQUEST_H_