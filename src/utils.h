#ifndef _UTILS_H_
#define _UTILS_H_

#include <algorithm>
#include <errno.h>
#include <stdlib.h>
#include <string.h>
#include <string>
#include <system_error>

inline char* memdup(const char* src, size_t size) {
    char* dest = (char*)malloc(size);
    if (dest == NULL)
        throw std::system_error(errno, std::system_category(), "Could not instantiate memory in memdup");
    memcpy(dest, src, size);
    return dest;
}

inline std::string trim(const std::string& s) {
    auto wsfront = std::find_if_not(s.begin(), s.end(), [](int c) { return std::isspace(c); });
    auto wsback = std::find_if_not(s.rbegin(), s.rend(), [](int c) { return std::isspace(c); }).base();
    return (wsback <= wsfront ? std::string() : std::string(wsfront, wsback));
}

#endif // !_UTILS_H_