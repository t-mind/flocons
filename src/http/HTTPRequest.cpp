#include "HTTPRequest.h"
#include "../Logger.h"
#include <curl/curl.h>

namespace flocons {

static auto logger = Logger::category("http");

HTTPRequest::~HTTPRequest() {
    if (this->_handle != NULL)
        curl_easy_cleanup((CURL*)this->_handle);
}

size_t HTTPRequest::_data_cb(char* ptr, size_t size, size_t nb, void* data) {
    auto request = reinterpret_cast<HTTPRequest*>(data);
    size *= nb;
    request->_response_buffer_size += size;
    if (request->_response_buffer_size > request->_response_buffer_max_size) {
        request->_response_buffer_max_size = request->_response_buffer_size;
        request->_response_buffer = (char*)realloc(request->_response_buffer, request->_response_buffer_size);
    }
    memcpy(request->_response_buffer + request->_response_buffer_size - size, ptr, size);
    return size;
}

size_t HTTPRequest::_header_cb(char* ptr, size_t size, size_t nb, void* data) {
    auto request = reinterpret_cast<HTTPRequest*>(data);
    size *= nb;
    std::string header(ptr, ptr + size);
    auto semicol_pos = header.find_first_of(':');
    if (semicol_pos != std::string::npos) {
        auto name = trim(header.substr(0, semicol_pos));
        auto value = trim(header.substr(semicol_pos + 1));
        logger->debug << "Read header " << name << " of value " << value << std::endl;
        if (name == HTTP_HEADER_LOCATION)
            request->_location = value;
        else if (name == HTTP_HEADER_CONTENT_TYPE)
            request->_mime_type = value;
        else if (name == HTTP_HEADER_FILE_MODE)
            sscanf(value.c_str(), "%o", &request->_mode);
        else if (name == HTTP_HEADER_LAST_MODIFIED)
            curl_getdate(value.c_str(), &request->_modification_time);
    }
    return size;
}

void HTTPRequest::_reset() {
    this->_response_buffer_size = 0;
    bzero(this->_error_buffer, 256);
    this->_location.clear();
    this->_mime_type.clear();
    this->_mode = 0;
    this->_modification_time = 0;

    CURL* curl;
    if (this->_handle == NULL) {
        curl = curl_easy_init();
        this->_handle = curl;
    } else {
        curl = (CURL*)this->_handle;
        curl_easy_reset(curl);
    }

    curl_easy_setopt(curl, CURLOPT_ERRORBUFFER, this->_error_buffer);
    curl_easy_setopt(curl, CURLOPT_FOLLOWLOCATION, 0L);
    curl_easy_setopt(curl, CURLOPT_WRITEFUNCTION, &HTTPRequest::_data_cb);
    curl_easy_setopt(curl, CURLOPT_WRITEDATA, this);
    curl_easy_setopt(curl, CURLOPT_HEADERFUNCTION, &HTTPRequest::_header_cb);
    curl_easy_setopt(curl, CURLOPT_HEADERDATA, this);
}

bool HTTPRequest::try_lock() {
    if (this->_mutex.try_lock()) {
        this->_locked = true;
        return true;
    }
    return false;
}

void HTTPRequest::unlock() {
    this->_locked = false;
    this->_mutex.unlock();
}

HTTPCode HTTPRequest::head(const std::string& relative_url) {
    this->_reset();
    auto full_url = this->_base_url / relative_url;
    logger->debug << "HEAD URL " << full_url << std::endl;
    CURL* curl = (CURL*)this->_handle;
    curl_easy_setopt(curl, CURLOPT_URL, (this->_base_url / relative_url).c_str());
    curl_easy_setopt(curl, CURLOPT_NOBODY, 1L);
    auto result = curl_easy_perform(curl);
    if (result != CURLE_OK)
        throw std::system_error(result, std::generic_category(), this->_error_buffer);

    long http_code;
    curl_easy_getinfo(curl, CURLINFO_RESPONSE_CODE, &http_code);
    return (HTTPCode)http_code;
}

HTTPCode HTTPRequest::get(const std::string& relative_url) {
    this->_reset();
    auto full_url = this->_base_url / relative_url;
    logger->debug << "GET URL " << full_url << std::endl;
    CURL* curl = (CURL*)this->_handle;
    curl_easy_setopt(curl, CURLOPT_URL, (this->_base_url / relative_url).c_str());
    auto result = curl_easy_perform(curl);
    if (result != CURLE_OK)
        throw std::system_error(result, std::generic_category(), this->_error_buffer);

    long http_code;
    curl_easy_getinfo(curl, CURLINFO_RESPONSE_CODE, &http_code);
    return (HTTPCode)http_code;
}

HTTPCode HTTPRequest::put(const std::string& relative_url, const char* data, size_t size, const std::string& mime_type) {
    this->_reset();
    auto full_url = this->_base_url / relative_url;
    logger->debug << "PUT URL " << full_url << std::endl;
    CURL* curl = (CURL*)this->_handle;

    curl_easy_setopt(curl, CURLOPT_URL, (this->_base_url / relative_url).c_str());
    curl_easy_setopt(curl, CURLOPT_CUSTOMREQUEST, "PUT");

    if (data != NULL) {
        curl_easy_setopt(curl, CURLOPT_POSTFIELDS, data);
        curl_easy_setopt(curl, CURLOPT_POSTFIELDSIZE, size);
    }

    struct curl_slist* headers = NULL;
    if (!mime_type.empty()) {
        auto header = "Content-Type: " + mime_type;
        headers = curl_slist_append(headers, header.c_str());
        curl_easy_setopt(curl, CURLOPT_HTTPHEADER, headers);
    }

    auto result = curl_easy_perform(curl);
    if (headers != NULL)
        curl_slist_free_all(headers);

    if (result != CURLE_OK)
        throw std::system_error(result, std::generic_category(), this->_error_buffer);

    long http_code;
    curl_easy_getinfo(curl, CURLINFO_RESPONSE_CODE, &http_code);
    return (HTTPCode)http_code;
}

} // namespace flocons