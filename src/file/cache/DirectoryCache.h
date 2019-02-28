#ifndef _FLOCONS_DIRECTORY_CACHE_H_
#define _FLOCONS_DIRECTORY_CACHE_H_

#include "../../common.h"

#define MAX_FOLDER_CACHE_SIZE 1000
#define FOLDER_CACHE_TO_PURGE_RATIO 0.20

namespace flocons {

class Directory;

class DirectoryCache {
  private:
    std::unordered_map<std::string, std::shared_ptr<Directory>> _cache;

  public:
    DirectoryCache() {}
    void clear();
    std::shared_ptr<Directory> get(const std::string& path);
    void set(const std::string& path, std::shared_ptr<Directory> directory);
};

} // namespace flocons

#include "../Directory.h"

#endif // !_FLOCONS_DIRECTORY_CACHE_H_