#include "FuseServer.h"
#include "../Logger.h"
#include <fuse.h>
#include <unistd.h>

namespace flocons {

static std::shared_ptr<FileService> file_service;
static thread_local auto logger = Logger::category("fuse");

static int do_getattr(const char* path, struct stat* st) {
    auto file = file_service->getFile(path);
    st->st_uid = getuid();
    st->st_gid = getgid();
    st->st_atime = file->modificationTime();
    st->st_mtime = file->modificationTime();
    st->st_mode = file->mode();

    switch (file->type()) {
    case File::Type::DIRECTORY:
        st->st_mode |= S_IFDIR;
        st->st_nlink = 2;
        break;
    case File::Type::REGULAR_FILE:
        auto regular_file = std::dynamic_pointer_cast<RegularFile>(file);
        st->st_mode |= S_IFREG;
        st->st_nlink = 1;
        st->st_size = regular_file->size();
        break;
    }
    return 0;
}

static int do_readdir(const char* path, void* buffer, fuse_fill_dir_t filler, off_t offset, struct fuse_file_info* fi) {
    auto files = file_service->listFiles(path);
    filler(buffer, ".", NULL, 0);
    filler(buffer, "..", NULL, 0);
    for (auto it = files.begin(); it != files.end(); ++it) { filler(buffer, (*it)->name().c_str(), NULL, 0); }
    return 0;
}

static int do_read(const char* path, char* buffer, size_t size, off_t offset, struct fuse_file_info* fi) { return 0; }

static int do_mkdir(const char* path, mode_t mode) {
    try {
        file_service->createDirectory(path, mode);
    } catch (std::system_error e) {
        logger->error << "Error while creating directory " << path << ": " << e.what() << " (" << e.code().value() << ")" << std::endl;
        return -e.code().value();
    }
    return 0;
}

int FuseServer::run(const std::vector<std::string> args) {
    char** argv = new char*[args.size()];
    int argc = 0;
    for (auto it = args.begin(); it != args.end(); ++it) {
        argv[argc] = (char*)malloc(it->length() + 1);
        strcpy(argv[argc], it->c_str());
        ++argc;
    }
    this->run(argc, argv);
}

int FuseServer::run(int argc, char* argv[]) {
    struct fuse_operations operations;
    bzero(&operations, sizeof(operations));

    file_service = this->_file_service;
    operations.getattr = &do_getattr;

    return fuse_main(argc, argv, &operations);
}

} // namespace flocons