#include "URL.h"

namespace flocons {

URL::URL(const URL& url) : _full_url(url._full_url), _hostname(url._hostname), _protocol(url._protocol), _port(url._port), _parts(url._parts) {}

URL::URL(const URL&& url)
    : _full_url(std::move(url._full_url)), _hostname(std::move(url._hostname)), _protocol(std::move(url._protocol)), _port(url._port),
      _parts(std::move(url._parts)) {}

URL::URL(const std::string& url) { this->_initialize_url(url.c_str()); }

URL::URL(const char* url) { this->_initialize_url(url); }

URL& URL::operator=(const URL& url) {
    this->_full_url = url._full_url;
    this->_hostname = url._hostname;
    this->_protocol = url._protocol;
    this->_port = url._port;
    this->_parts = url._parts;
    return *this;
}

URL& URL::operator=(const URL&& url) {
    this->_full_url = std::move(url._full_url);
    this->_hostname = std::move(url._hostname);
    this->_protocol = std::move(url._protocol);
    this->_port = url._port;
    this->_parts = std::move(url._parts);
    return *this;
}

URL& URL::operator=(const std::string& url) {
    this->_initialize_url(url.c_str());
    return *this;
}

URL& URL::operator=(const char* url) {
    this->_initialize_url(url);
    return *this;
}

void URL::_initialize_url(const char* url) {
    if (this->_parts.size() > 0) this->_parts.clear();
    static __thread char protocol[8];
    static __thread char hostname[256];
    static __thread int port;

    char* ptr = (char*)url;
    int match;
    if ((match = sscanf(url, "%[^:]://%[^:/]:%d", protocol, hostname, &port)) >= 2) {
        this->_hostname = hostname;
        this->_protocol = protocol;
        ptr += this->_protocol.length() + this->_hostname.length() + 3;
        if (match == 2) {
            if (this->_protocol == "https")
                this->_port = 443;
            else
                this->_port = 80;
        } else {
            this->_port = port;
            ptr += std::to_string(port).length() + 1;
        }
    } else {
        throw std::logic_error(std::string("Malformed url ") + url);
    }
    this->_full_url.assign(url, ptr - url);
    this->_append_relative_url(ptr);
}

void URL::_append_relative_url(const char* url) {
    char* buffer = strdup(url);
    char* ptr = buffer;

    while (char* token = strtok_r(ptr, "/", &ptr)) {
        this->_full_url += "/";
        this->_full_url += token;
        this->_parts.push_back(token);
    }

    free(buffer);
}

URL URL::operator/(const std::string& url) const {
    URL final_url(*this);
    final_url /= url;
    return final_url;
}

URL URL::operator/(const char* url) const {
    URL final_url(*this);
    final_url /= url;
    return final_url;
}

void URL::operator/=(const std::string& url) { this->_append_relative_url(url.c_str()); }

void URL::operator/=(const char* url) { this->_append_relative_url(url); }

bool URL::isValid(const std::string& candidate) {
    try {
        URL test(candidate);
    } catch (std::logic_error e) { return false; }
    return true;
}

} // namespace flocons