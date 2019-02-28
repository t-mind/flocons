#include "HTTPFileServer.h"
#include "../Logger.h"
#include "../file/serialize/HTMLTableFileSerializer.h"
#include "../http/http.h"
#include <microhttpd.h>

namespace flocons {

static void addReponseHeader(struct MHD_Response* response, const std::shared_ptr<File> file) {
    if (file->type() == File::Type::REGULAR_FILE) {
        auto regular_file = std::dynamic_pointer_cast<RegularFile>(file);
        auto size_str = std::to_string(regular_file->size());
        MHD_add_response_header(response, HTTP_HEADER_CONTENT_LENGTH, size_str.c_str());
    }
    MHD_add_response_header(response, HTTP_HEADER_CONTENT_TYPE, file->mimeType().c_str());

    static __thread char mode_string[5];
    sprintf(mode_string, "%o", file->mode());
    MHD_add_response_header(response, HTTP_HEADER_FILE_MODE, mode_string);

    static __thread char time_string[256];
    static __thread struct tm tm;
    time_t time = file->modificationTime();
    gmtime_r(&time, &tm);
    strftime(time_string, sizeof time_string, "%a, %d %b %Y %H:%M:%S %Z", &tm);
    MHD_add_response_header(response, HTTP_HEADER_LAST_MODIFIED, time_string);
}

static MHD_Response* getFileInfo(const std::shared_ptr<FileService>& file_service, const char* url) {
    auto file = file_service->getFile(url);
    if (file != nullptr) {
        auto response = MHD_create_response_from_buffer(0, NULL, MHD_RESPMEM_PERSISTENT);
        addReponseHeader(response, file);
        return response;
    }
    return NULL;
}

static MHD_Response* getFileData(const std::shared_ptr<FileService>& file_service, const char* url, const std::string& accept_types) {
    auto file = file_service->getFile(url);
    struct MHD_Response* response = NULL;
    switch (file->type()) {
    case File::Type::REGULAR_FILE: {
        auto regular_file = std::dynamic_pointer_cast<RegularFile>(file);
        response = MHD_create_response_from_buffer(regular_file->size(), regular_file->data(), MHD_RESPMEM_MUST_FREE);
        addReponseHeader(response, regular_file);
        break;
    }
    case File::Type::DIRECTORY: {
        auto directory = std::dynamic_pointer_cast<Directory>(file);
        auto files = directory->listFiles();
        char* buffer = NULL;
        size_t size = 0;
        FileSerializer::write<HTMLTableFileSerializer>(files, &buffer, &size);
        response = MHD_create_response_from_buffer(size, buffer, MHD_RESPMEM_MUST_FREE);
        addReponseHeader(response, directory);
        break;
    }
    }
    return response;
}

static int process_request(void* cls, struct MHD_Connection* connection, const char* url, const char* method, const char* version, const char* upload_data,
                           size_t* upload_data_size, void** ptr) {

    if (*ptr == NULL) // New request
        Logger::debug << "Process URL " << url << " with method " << method << std::endl;

    struct MHD_Response* response = NULL;
    int return_code = MHD_HTTP_OK;
    auto file_service = *reinterpret_cast<std::shared_ptr<FileService>*>(cls);

    try {
        if (strcmp(method, MHD_HTTP_METHOD_HEAD) == 0) {
            if (*ptr == NULL) {
                // This is a new request
                static int dummy;
                *ptr = &dummy;
                return MHD_YES;
            }
            response = getFileInfo(file_service, url);
        } else if (strcmp(method, MHD_HTTP_METHOD_GET) == 0) {
            if (*ptr == NULL) {
                // This is a new request
                static int dummy;
                *ptr = &dummy;
                return MHD_YES;
            }
            const char* accept = MHD_lookup_connection_value(connection, MHD_HEADER_KIND, MHD_HTTP_HEADER_ACCEPT);
            response = getFileData(file_service, url, accept);
        } else if (strcmp(method, MHD_HTTP_METHOD_PUT) == 0) {
            if (*ptr == NULL) {
                // This is a new request
                *ptr = new std::vector<char>();
                return MHD_YES;
            }
            auto buffer = (std::vector<char>*)*ptr;
            if (*upload_data_size) {
                buffer->insert(buffer->end(), upload_data, upload_data + *upload_data_size);
                *upload_data_size = 0;
                return MHD_YES;
            } else {
                const char* content_type = MHD_lookup_connection_value(connection, MHD_HEADER_KIND, MHD_HTTP_HEADER_CONTENT_TYPE);
                if (strcmp(content_type, "inode/directory") == 0) {
                    file_service->createDirectory(url);
                } else {
                    file_service->createRegularFile(url, buffer->data(), buffer->size());
                }
                response = MHD_create_response_from_buffer(0, NULL, MHD_RESPMEM_PERSISTENT);
                delete buffer;
                *ptr = NULL;
            }
        } else {
            return_code = MHD_HTTP_BAD_REQUEST;
            response = MHD_create_response_from_buffer(0, NULL, MHD_RESPMEM_PERSISTENT);
        }
    } catch (std::system_error e) {
        Logger::debug << "Error while parsing request " << e.what() << " - " << e.code().value() << std::endl;
        if (e.code().value() == ENOENT)
            return_code = MHD_HTTP_NOT_FOUND;
        else
            return_code = MHD_HTTP_INTERNAL_SERVER_ERROR;
        response = MHD_create_response_from_buffer(strlen(e.what()), (char*)e.what(), MHD_RESPMEM_MUST_COPY);
    }

    if (response == NULL) {
        return_code = MHD_HTTP_INTERNAL_SERVER_ERROR;
        static const char* error_message = "No reponse given";
        MHD_create_response_from_buffer(strlen(error_message), (char*)error_message, MHD_RESPMEM_PERSISTENT);
    }

    Logger::debug << "Answer with code " << return_code << std::endl;
    int ret = MHD_queue_response(connection, return_code, response);
    MHD_destroy_response(response);
    return ret;
}

void HTTPFileServer::start() {
    Logger::debug << "Start HTTP File server on port " << this->_port << std::endl;
    unsigned int flags = MHD_USE_INTERNAL_POLLING_THREAD;
    if (MHD_is_feature_supported(MHD_FEATURE_EPOLL))
        flags |= MHD_USE_EPOLL;
    else if (MHD_is_feature_supported(MHD_FEATURE_POLL))
        flags |= MHD_USE_POLL;

    this->_daemon = MHD_start_daemon(flags, this->_port, NULL, NULL, &process_request, &this->_file_service, MHD_OPTION_END);
    if (this->_daemon == NULL) { throw std::system_error(errno, std::system_category(), "Could not start http server"); }
}

void HTTPFileServer::stop() {
    if (this->_daemon != NULL) {
        Logger::debug << "Stop HTTP File server on port " << this->_port << std::endl;
        MHD_stop_daemon((struct MHD_Daemon*)this->_daemon);
        this->_daemon = NULL;
    }
}

} // namespace flocons