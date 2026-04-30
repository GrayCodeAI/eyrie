package constants

// Image limits
const (
	APIImageMaxBase64Size       = 5 * 1024 * 1024        // 5 MB
	ImageTargetRawSize          = APIImageMaxBase64Size * 3 / 4 // 3.75 MB
	ImageMaxWidth               = 2000
	ImageMaxHeight              = 2000
)

// PDF limits
const (
	PDFTargetRawSize            = 20 * 1024 * 1024       // 20 MB
	APIPDFMaxPages              = 100
	PDFExtractSizeThreshold     = 3 * 1024 * 1024        // 3 MB
	PDFMaxExtractSize           = 100 * 1024 * 1024      // 100 MB
	PDFMaxPagesPerRead          = 20
	PDFAtMentionInlineThreshold = 10
)

// Media limits
const (
	APIMaxMediaPerRequest = 100
)
