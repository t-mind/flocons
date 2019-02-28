#ifndef _FLOCONS_LOCAL_CONTEXT_H_
#define _FLOCONS_LOCAL_CONTEXT_H_

#include "../file/Path.h"
#include "../file/cache/DirectoryCache.h"

namespace flocons {

class LocalContext {
  public:
    enum FileMode { local, remote };

  private:
    std::string _name;
    Path _path;
    DirectoryCache _cache;

  public:
    LocalContext(const std::string& name, const Path& path) : _name(name), _path(path) {}
    const Path& path() const { return this->_path; }
    const std::string& name() const { return this->_name; }

    std::shared_ptr<Directory> getDirectoryFromCache(const std::string& path) { return this->_cache.get(path); }
    std::shared_ptr<Directory> getDirectoryFromCache(const Path& path) { return this->_cache.get(path.string()); }
    void addDirectoryToCache(const std::string& path, std::shared_ptr<Directory> directory) { this->_cache.set(path, directory); }
    void addDirectoryToCache(const Path& path, std::shared_ptr<Directory> directory) { this->_cache.set(path.string(), directory); }
};

} // namespace flocons

#endif // !_FLOCONS_LOCAL_CONTEXT_H_
