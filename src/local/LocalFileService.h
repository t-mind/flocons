#ifndef _FLOCONS_LOCAL_FILE_SERVICE_H_
#define _FLOCONS_LOCAL_FILE_SERVICE_H_

#include "../common.h"
#include "../file/Directory.h"
#include "../file/File.h"
#include "../file/FileService.h"
#include "../file/Path.h"
#include "../file/RegularFile.h"
#include "LocalContext.h"

namespace flocons {

class LocalFileService : public FileService {
  private:
    std::shared_ptr<LocalContext> _context;
    std::shared_ptr<Directory> _root_directory;

  public:
    LocalFileService(const std::string& name, const Path& path);

    virtual std::shared_ptr<File> getFile(const Path& path);
    virtual std::shared_ptr<Directory> createDirectory(const Path& path, mode_t mode = 0755);
    virtual std::shared_ptr<RegularFile> createRegularFile(const Path& path, const char* data, size_t size, mode_t mode = 0644);
    virtual std::vector<std::shared_ptr<File>> listFiles(const Path& path);
};

} // namespace flocons

#endif // !_FLOCONS_LOCAL_FILE_SERVICE_H_