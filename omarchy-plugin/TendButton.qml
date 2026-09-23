import qs.Ui

Button {
  property bool checkable: false
  property bool checked: false

  focusable: true
  bordered: true
  selected: checkable && checked
}
