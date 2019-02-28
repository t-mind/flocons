#ifndef _FLOCONS_FILE_PATH_H_
#define _FLOCONS_FILE_PATH_H_

#include "../common.h"

#ifdef WIN32
#define SEPARATOR "\\"
#else
#define SEPARATOR "/"
#endif

namespace flocons {

class Path {
  private:
    std::string _full_path;
    std::string _dirname;
    std::string _extension;
    std::vector<std::string> _parts;
    void _initialize_path(const char* path);
    void _append_path(const char* path);

  public:
    Path();
    Path(const Path& path);
    Path(const Path&& path);
    Path(const std::string& path);
    Path(const char* path);
    Path& operator=(const Path& path);
    Path& operator=(const Path&& path);
    Path& operator=(const std::string& path);
    Path& operator=(const char* path);
    Path operator/(const Path& subpath) const;
    Path operator/(const std::string& subpath) const;
    Path operator/(const char* subpath) const;
    void operator/=(const Path& subpath);
    void operator/=(const std::string& subpath);
    void operator/=(const char* subpath);
    bool operator==(const Path& path) const { return this->_full_path == path._full_path; }
    bool operator==(const char* path) const { return this->_full_path == path; }
    bool operator==(const std::string& path) const { return this->_full_path == path; }

    const std::string& extension() const { return this->_extension; }
    const std::string& basename() const;
    const std::string& dirname() const { return this->_dirname; }
    const std::string& string() const { return this->_full_path; }
    const char* c_str() const { return this->_full_path.c_str(); }
    std::vector<std::string>::const_iterator begin() const { return this->_parts.begin(); }
    std::vector<std::string>::const_iterator end() const { return this->_parts.end(); }
    std::vector<std::string>::const_reverse_iterator rbegin() const { return this->_parts.rbegin(); }
    std::vector<std::string>::const_reverse_iterator rend() const { return this->_parts.rend(); }
};

// string and stream operators
inline std::ostream& operator<<(std::ostream& stream, const Path& path) {
    stream << path.string();
    return stream;
}
inline std::string operator+(const std::string& s, const Path& p) { return s + p.string(); }
inline std::string operator+(const Path& p, const std::string& s) { return p.string() + s; }
inline std::string operator+(const char* s, const Path& p) { return s + p.string(); }
inline std::string operator+(const Path& p, const char* s) { return p.string() + s; }

} // namespace flocons

#endif // !_FLOCONS_FILE_PATH_H_