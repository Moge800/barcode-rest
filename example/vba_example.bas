Attribute VB_Name = "vba_example"
Option Explicit

Private Const API_BASE As String = "http://127.0.0.1:8787"

Public Sub PutDataMatrixExample()
    Call PutBarcode( _
        Range("A1"), _
        Range("A2"), _
        150, _
        "datamatrix", _
        "&size=512")
End Sub

Public Sub PutQRCodeExample()
    Call PutBarcode( _
        Range("A1"), _
        Range("A10"), _
        150, _
        "qr", _
        "&size=512&level=M")
End Sub

Public Sub PutCode128Example()
    Call PutBarcode( _
        Range("A1"), _
        Range("A18"), _
        250, _
        "code128", _
        "&module=3&height=80&label=1")
End Sub

Public Sub PutBarcode( _
    ByVal sourceCell As Range, _
    ByVal imageCell As Range, _
    ByVal imageWidth As Double, _
    ByVal barcodeType As String, _
    ByVal options As String)

    Dim text As String
    Dim url As String
    Dim tempFile As String
    Dim shapeName As String

    Dim http As Object
    Dim stream As Object
    Dim picture As Shape
    Dim placementArea As Range
    Dim targetSheet As Worksheet
    Dim errorMessage As String

    On Error GoTo ErrorHandler

    text = CStr(sourceCell.Value2)

    If Len(text) = 0 Then
        MsgBox "The source cell is empty.", vbExclamation
        Exit Sub
    End If

    If imageWidth <= 0 Then
        MsgBox "The image width must be greater than zero.", vbExclamation
        Exit Sub
    End If

    url = API_BASE & "/" & barcodeType _
        & "?text=" & Application.WorksheetFunction.EncodeURL(text) _
        & options

    Set http = CreateObject("MSXML2.XMLHTTP.6.0")
    http.Open "GET", url, False
    http.Send

    If http.Status <> 200 Then
        Err.Raise vbObjectError + 1000, , _
            "barcode-rest returned an error." & vbCrLf _
            & "HTTP " & http.Status & vbCrLf _
            & http.responseText
    End If

    tempFile = Environ$("TEMP") _
        & "\barcode-rest_" _
        & Format$(Now, "yyyymmdd_hhnnss") _
        & "_" & CStr(CLng(Timer * 1000)) _
        & ".png"

    Set stream = CreateObject("ADODB.Stream")
    stream.Type = 1
    stream.Open
    stream.Write http.responseBody
    stream.SaveToFile tempFile, 2
    stream.Close

    Set targetSheet = imageCell.Worksheet
    Set placementArea = imageCell.MergeArea

    shapeName = "barcode_" _
        & Replace(placementArea.Address(False, False), ":", "_")

    ' Remove an existing barcode at the same placement cell.
    On Error Resume Next
    targetSheet.Shapes(shapeName).Delete
    On Error GoTo ErrorHandler

    Set picture = targetSheet.Shapes.AddPicture( _
        Filename:=tempFile, _
        LinkToFile:=msoFalse, _
        SaveWithDocument:=msoTrue, _
        Left:=placementArea.Left, _
        Top:=placementArea.Top, _
        Width:=-1, _
        Height:=-1)

    With picture
        .Name = shapeName
        .LockAspectRatio = msoTrue
        .Placement = xlMove
        .Width = imageWidth

        ' Align the image's top-left corner with the placement cell.
        .Left = placementArea.Left
        .Top = placementArea.Top

        .AlternativeText = "Barcode: " & text
    End With

    Kill tempFile
    Exit Sub

ErrorHandler:
    errorMessage = Err.Description

    On Error Resume Next

    If Not stream Is Nothing Then stream.Close

    If Len(tempFile) > 0 Then
        If Len(Dir$(tempFile)) > 0 Then Kill tempFile
    End If

    MsgBox errorMessage, vbCritical, "Barcode generation error"
End Sub
