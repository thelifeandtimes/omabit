import qs.Ui
import qs.Commons

Button {
  property bool checkable: false
  property bool checked: false

  implicitHeight: Style.spacing.controlHeight
  focusable: true
  bordered: true
  selected: checkable && checked
}
