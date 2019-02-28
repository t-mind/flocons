#ifndef _FLOCONS_FILE_SERIALIZER_H_
#define _FLOCONS_FILE_SERIALIZER_H_

#include "../../common.h"
#include "../File.h"

#ifndef __GNUC__
#define __attribute__(a)
#endif

namespace flocons {
class FileSerializer {
  public:
    enum class StreamOperation { flush };
    class IStream {
      private:
        std::unique_ptr<FileSerializer> _serializer;

      public:
        IStream(std::unique_ptr<FileSerializer>&& serializer) : _serializer(std::move(serializer)) {}
        size_t operator>>(std::shared_ptr<File>& file) { return this->_serializer->read(file); }
        size_t operator>>(std::vector<std::shared_ptr<File>>& files) { return this->_serializer->read(files); }
        void close() { this->_serializer->close(); }
    };
    class OStream {
      private:
        std::unique_ptr<FileSerializer> _serializer;

      public:
        OStream(std::unique_ptr<FileSerializer>&& serializer) : _serializer(std::move(serializer)) {}
        OStream& operator<<(const std::shared_ptr<File>& file) {
            this->_serializer->write(file);
            return *this;
        }
        OStream& operator<<(const std::vector<std::shared_ptr<File>>& files) {
            this->_serializer->write(files);
            return *this;
        }
        OStream& operator<<(StreamOperation op) {
            if (op == StreamOperation::flush) this->_serializer->flush();
            return *this;
        }

        void close() { this->_serializer->close(); }
    };
    friend class IStream;
    friend class OStream;

  private:
    FILE* _file = NULL;
    size_t _max_file_size = 0;
    size_t _file_initial_position = 0;
    bool _must_close_file = true;

  protected:
    enum Mode { INPUT, OUTPUT };
    Mode _mode;
    size_t _processed = 0;
    bool _closed = false;
    bool _error = false;

  public:
    FileSerializer(Mode mode, char* dest, size_t max_size);
    FileSerializer(Mode mode, char** dest, size_t* size);
    FileSerializer(Mode mode, FILE* file, bool must_close_file);
    virtual ~FileSerializer();

    virtual size_t read(std::shared_ptr<File>& file) = 0;
    virtual size_t read(std::vector<std::shared_ptr<File>>& files);
    virtual size_t write(const std::shared_ptr<File>& file) = 0;
    virtual size_t write(const std::vector<std::shared_ptr<File>>& files);

    int readFormattedLine(const char* format, ...) __attribute__((format(scanf, 2, 3)));
    size_t writeFormat(const char* format, ...) __attribute__((format(printf, 2, 3)));
    void flush();
    virtual void close();

    size_t processed() const { return this->_processed; }

    static std::unique_ptr<IStream> read();

    template <class S> static std::unique_ptr<IStream> reader(FILE* file) { return std::make_unique<IStream>(std::make_unique<S>(Mode::INPUT, file)); }

    template <class S> static std::unique_ptr<OStream> writer(char* dest, int max_size) {
        return std::make_unique<OStream>(std::make_unique<S>(Mode::OUTPUT, dest, max_size));
    }
    template <class S> static std::unique_ptr<OStream> writer(char** dest, size_t* size) {
        return std::make_unique<OStream>(std::make_unique<S>(Mode::OUTPUT, dest, size));
    }
    template <class S> static std::unique_ptr<OStream> writer(FILE* file, bool must_close_file) {
        return std::make_unique<OStream>(std::make_unique<S>(Mode::OUTPUT, file, must_close_file));
    }

    template <class S> static size_t read(FILE* file, std::vector<std::shared_ptr<File>>& files, bool must_close_file) {
        S s(Mode::INPUT, file, must_close_file);
        s.read(files);
        s.close();
        return s.processed();
    }

    template <class S> static size_t write(const std::vector<std::shared_ptr<File>>& files, char** dest, size_t* size) {
        S s(Mode::OUTPUT, dest, size);
        s.write(files);
        s.close();
        return s.processed();
    }
};

inline FileSerializer::OStream& operator<<(std::unique_ptr<FileSerializer::OStream>& stream, const std::vector<std::shared_ptr<File>>& files) {
    return *stream << files;
}
inline FileSerializer::OStream& operator<<(std::unique_ptr<FileSerializer::OStream>& stream, const std::shared_ptr<File>& file) { return *stream << file; }

} // namespace flocons

#endif // !_FLOCONS_FILE_SERIALIZER_H_