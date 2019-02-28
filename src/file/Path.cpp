#include "Path.h"

namespace flocons {

Path::Path() : _full_path("/"), _dirname("/"), _extension("") {}

Path::Path(const Path& path) : _full_path(path._full_path), _parts(path._parts) {}

Path::Path(const Path&& path) : _full_path(std::move(path._full_path)), _parts(std::move(path._parts)) {}

Path::Path(const std::string& path) { this->_initialize_path(path.c_str()); }

Path::Path(const char* path) { this->_initialize_path(path); }

Path& Path::operator=(const Path& path) {
    this->_full_path = path._full_path;
    this->_parts = path._parts;
    return *this;
}

Path& Path::operator=(const Path&& path) {
    this->_full_path = std::move(path._full_path);
    this->_parts = std::move(path._parts);
    return *this;
}

Path& Path::operator=(const std::string& path) {
    this->_initialize_path(path.c_str());
    return *this;
}

Path& Path::operator=(const char* path) {
    this->_initialize_path(path);
    return *this;
}

void Path::_initialize_path(const char* path) {
    if (this->_parts.size() > 0) this->_parts.clear();
    this->_full_path = "/";
    this->_dirname = "/";
    this->_append_path(path);
}

void Path::_append_path(const char* path) {
    char* buffer = strdup(path);
    char* ptr = buffer;

    while (char* token = strtok_r(ptr, SEPARATOR, &ptr)) {
        this->_dirname = this->_full_path; // Save last full path as 'dirname'

        if (this->_parts.size() > 0) this->_full_path += SEPARATOR;
        this->_full_path += token;
        this->_parts.push_back(token);
    }

    // extract extension
    if (this->_parts.size() > 0) {
        const std::string& last = this->_parts.back();
        auto point_position = last.find_last_of('.');
        this->_extension = point_position != std::string::npos ? last.substr(point_position + 1) : "";
    } else {
        this->_extension = "";
    }

    free(buffer);
}

Path Path::operator/(const Path& path) const {
    Path final_path(*this);
    final_path /= path;
    return final_path;
}

Path Path::operator/(const std::string& path) const {
    Path final_path(*this);
    final_path /= path;
    return final_path;
}

Path Path::operator/(const char* path) const {
    Path final_path(*this);
    final_path /= path;
    return final_path;
}

void Path::operator/=(const Path& path) {
    this->_full_path += path._full_path;
    this->_parts.reserve(this->_parts.size() + path._parts.size());
    this->_parts.insert(this->_parts.end(), path._parts.begin(), path._parts.end());
    this->_extension = path._extension;
}

void Path::operator/=(const std::string& path) { this->_append_path(path.c_str()); }

void Path::operator/=(const char* path) { this->_append_path(path); }

const std::string& Path::basename() const {
    if (this->_parts.size() > 0) return this->_parts.back();

    return this->_full_path; // Because if there is no parts, _full_path contains '/'
}

} // namespace flocons