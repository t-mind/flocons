#ifndef _FLOCONS_HTTP_URL_H_
#define _FLOCONS_HTTP_URL_H_

#include "../common.h"

namespace flocons {

class URL {
  private:
    std::string _full_url;
    std::string _hostname;
    std::string _protocol;
    int _port;
    std::vector<std::string> _parts;
    void _initialize_url(const char* url);
    void _append_relative_url(const char* relative_url);

  public:
    URL(const URL& url);
    URL(const URL&& url);
    URL(const std::string& url);
    URL(const char* url);
    URL& operator=(const URL& url);
    URL& operator=(const URL&& url);
    URL& operator=(const std::string& url);
    URL& operator=(const char* url);
    URL operator/(const std::string& suburl) const;
    URL operator/(const char* suburl) const;
    void operator/=(const std::string& suburl);
    void operator/=(const char* suburl);
    bool operator==(const URL& url) const { return this->_full_url == url._full_url; }
    bool operator==(const char* url) const { return this->_full_url == url; }
    bool operator==(const std::string& url) const { return this->_full_url == url; }

    const std::string& hostname() const { return this->_hostname; }
    const std::string& protocol() const { return this->_protocol; }
    int port() const { return this->_port; }
    const std::string& string() const { return this->_full_url; }
    const char* c_str() const { return this->_full_url.c_str(); }

    std::vector<std::string>::const_iterator begin() const { return this->_parts.begin(); }
    std::vector<std::string>::const_iterator end() const { return this->_parts.end(); }
    std::vector<std::string>::const_reverse_iterator rbegin() const { return this->_parts.rbegin(); }
    std::vector<std::string>::const_reverse_iterator rend() const { return this->_parts.rend(); }

    static bool isValid(const std::string& candidate);
};

// string and stream operators
inline std::ostream& operator<<(std::ostream& stream, const URL& url) {
    stream << url.string();
    return stream;
}
inline std::string operator+(const std::string& s, const URL& p) { return s + p.string(); }
inline std::string operator+(const URL& p, const std::string& s) { return p.string() + s; }
inline std::string operator+(const char* s, const URL& p) { return s + p.string(); }
inline std::string operator+(const URL& p, const char* s) { return p.string() + s; }

} // namespace flocons

#endif // !_FLOCONS_HTTP_URL_H_