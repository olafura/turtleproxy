import QtQuick
import Qt.labs.platform


Window {
    width: 640
    height: 480
    visible: true
    title: qsTr("TurtleProxy")

    Image {
        id: turtleProxy
        x: 251
        y: 353
        source: "images/turtle-proxy.svg"
        fillMode: Image.PreserveAspectFit
    }
}
