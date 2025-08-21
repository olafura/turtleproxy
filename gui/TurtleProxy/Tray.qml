import QtQuick
import Qt.labs.platform

SystemTrayIcon {
    visible: true
    icon.source: "qrc:/images/turtle-proxy-white.png"

    menu: Menu {
        MenuItem {
            text: qsTr("Quit")
            onTriggered: Qt.quit()
        }
    }
}
