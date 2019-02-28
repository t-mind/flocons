#include "FileSerializer.h"
#include "../../Logger.h"
#include <stdarg.h>
#include <sys/stat.h>
#include <unistd.h>

namespace flocons {
static thread_local auto logger = Logger::category("serializer");

FileSerializer::FileSerializer(Mode mode, char* dest, size_t max_size)
    : _file(fmemopen(dest, max_size, mode == Mode::INPUT ? "r" : "w")), _max_file_size(max_size), _mode(mode) {}

FileSerializer::FileSerializer(Mode mode, char** dest, size_t* size) : _file(open_memstream(dest, size)), _mode(mode) {
    if (mode != Mode::OUTPUT) throw new std::logic_error("Can't open memory stream in read mode");
}

FileSerializer::FileSerializer(Mode mode, FILE* file, bool must_close_file)
    : _file(file), _mode(mode), _file_initial_position(ftell(file)), _must_close_file(must_close_file) {}

FileSerializer::~FileSerializer() { this->close(); }

size_t FileSerializer::read(std::vector<std::shared_ptr<File>>& files) {
    size_t last_processed = -1;
    while (last_processed != this->_processed) {
        last_processed = this->_processed;
        std::shared_ptr<File> file;
        this->read(file);
        if (file != nullptr) files.push_back(file);
    }
    return this->_processed;
}

size_t FileSerializer::write(const std::vector<std::shared_ptr<File>>& files) {
    for (auto it = files.begin(); it != files.end(); ++it) this->write(*it);
    return this->_processed;
}

size_t FileSerializer::writeFormat(const char* format, ...) {
    if (!this->_closed && !this->_error && this->_file != NULL) { // Ignore write following error
        va_list ap;
        va_start(ap, format);
        int written = vfprintf(this->_file, format, ap);
        if (written < 0) {
            this->_error = true;
            throw std::system_error(ferror(this->_file), std::system_category(), "Could not serialize file");
        }
        this->_processed += written;
        va_end(ap);
    }
    return this->_processed;
}

int FileSerializer::readFormattedLine(const char* format, ...) {
    int scanned = EOF;
    if (!this->_closed && !this->_error && this->_file != NULL) { // Ignore write following error
        static __thread char line[4096];
        va_list ap;
        va_start(ap, format);
        if (fgets(line, 4096, this->_file) != NULL) {
            int line_size = strlen(line);
            if (line[line_size - 1] != '\n') {
                // File not complete
                fseek(this->_file, -line_size, SEEK_CUR);
            } else {
                logger->debug << "Read line " << line << std::endl;
                scanned = vsscanf(line, format, ap);
                this->_processed += line_size + 1;
            }
        }
        va_end(ap);
    }
    return scanned;
}

void FileSerializer::close() {
    if (!this->_closed) {
        if (this->_file != NULL) {
            if (this->_must_close_file)
                fclose(this->_file);
            else
                this->flush();
        }
        this->_closed = true;
    }
}

void FileSerializer::flush() {
    if (!this->_closed && !this->_error && this->_file != NULL && this->_mode == Mode::OUTPUT) {
        fflush(this->_file);
        fsync(fileno(this->_file));
    }
}

} // namespace flocons