#include "DirectoryCache.h"

namespace flocons {

void DirectoryCache::clear() { this->_cache.clear(); }

std::shared_ptr<Directory> DirectoryCache::get(const std::string& path) {
    auto it = this->_cache.find(path);
    if (it == this->_cache.end())
        return nullptr;
    return it->second;
}

void DirectoryCache::set(const std::string& path, std::shared_ptr<Directory> directory) { this->_cache[path] = directory; }

} // namespace flocons