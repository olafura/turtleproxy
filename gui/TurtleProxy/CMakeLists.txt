cmake_minimum_required(VERSION 3.16)

project(TurtleProxy VERSION 0.1 LANGUAGES CXX)

set(CMAKE_AUTOMOC ON)
set(CMAKE_CXX_STANDARD_REQUIRED ON)

find_package(Qt6 REQUIRED COMPONENTS Quick)


qt_add_executable(appTurtleProxy
    main.cpp
)

qt_add_qml_module(appTurtleProxy
    URI TurtleProxy
    VERSION 1.0
    QML_FILES
        Main.qml
        QML_FILES
        QML_FILES Tray.qml
        RESOURCES icon.qrc
        RESOURCES images/turtle-proxy.svg
        RESOURCES images/turtle-proxy.png
        RESOURCES images/turtle-proxy-white.png
)

# Qt for iOS sets MACOSX_BUNDLE_GUI_IDENTIFIER automatically since Qt 6.1.
# If you are developing for iOS or macOS you should consider setting an
# explicit, fixed bundle identifier manually though.
set_target_properties(appTurtleProxy PROPERTIES
#    MACOSX_BUNDLE_GUI_IDENTIFIER com.example.appTurtleProxy
    MACOSX_BUNDLE_BUNDLE_VERSION ${PROJECT_VERSION}
    MACOSX_BUNDLE_SHORT_VERSION_STRING ${PROJECT_VERSION_MAJOR}.${PROJECT_VERSION_MINOR}
    MACOSX_BUNDLE TRUE
    WIN32_EXECUTABLE TRUE
)

target_link_libraries(appTurtleProxy
    PRIVATE Qt6::Quick
)

include(GNUInstallDirs)
install(TARGETS appTurtleProxy
    BUNDLE DESTINATION .
    LIBRARY DESTINATION ${CMAKE_INSTALL_LIBDIR}
    RUNTIME DESTINATION ${CMAKE_INSTALL_BINDIR}
)
