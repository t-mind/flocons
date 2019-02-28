#include "../Logger.h"
#include "../common.h"
#include "../fuse/FuseServer.h"
#include "../local/LocalFileService.h"
#include "../remote/HTTPFileService.h"
#include <getopt.h>
#include <unistd.h>

using namespace flocons;

void print_usage(char* program_name) {
    std::cout << program_name << " [--verbose|-v] [--version|-V] [--help|-h] [--hostname <hostname>] <source> <mounting_point>" << std::endl;
}

int main(int argc, char** argv) {
    int verbose_flag = 0;

    char hostname[HOST_NAME_MAX + 1];
    gethostname(hostname, HOST_NAME_MAX + 1);

    std::string source, mounting_point;
    std::shared_ptr<FileService> file_service;

    static struct option long_options[] = {{"verbose", no_argument, 0, 'v'},
                                           {"help", no_argument, 0, 'h'},
                                           {"version", no_argument, 0, 'V'},
                                           {"hostname", required_argument, 0, 'H'},
                                           {0, 0, 0, 0}};

    std::vector<std::string> fuse_args = {"-s"}; // multithread by default

    int arg_index = 1;
    int c;
    while ((c = getopt_long(argc, argv, "vVhH:", long_options, NULL)) != -1) {
        ++arg_index;
        Logger::debug << "Parse argument " << (char)c << " at pos " << arg_index << std::endl;
        if (optarg != NULL) {
            ++arg_index;
            Logger::debug << "Value is " << optarg << std::endl;
        }
        switch (c) {
        case 'H':
            if (strlen(optarg) > HOST_NAME_MAX)
                throw std::logic_error(std::string("Hostname ") + optarg + " is longer than " + std::to_string(HOST_NAME_MAX));
            strcpy(hostname, optarg);
            break;
        case 'v':
            verbose_flag = 1;
            fuse_args.push_back("-d");
            break;
        case 'V':
            fuse_args = {"-V"};
            goto start_fuse;
            break;
        case 'h':
            fuse_args = {"-h"};
            print_usage(argv[0]);
            goto start_fuse;
            break;
        case '?':
            print_usage(argv[0]);
            return 1;
        }
    }

    if (arg_index != argc - 2) {
        print_usage(argv[0]);
        return 1;
    }

    source = argv[arg_index];
    if (URL::isValid(source))
        file_service = std::make_shared<HTTPFileService>(source);
    else
        file_service = std::make_shared<LocalFileService>(hostname, source);

    mounting_point = argv[arg_index + 1];
    fuse_args.push_back(mounting_point);

start_fuse:
    FuseServer server(file_service);
    return server.run(fuse_args);
}