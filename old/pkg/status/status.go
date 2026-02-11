package status

// HTTP status codes following FastAPI conventions
const (
	// 1xx Informational
	Continue           = 100
	SwitchingProtocols = 101
	Processing         = 102
	EarlyHints         = 103

	// 2xx Success
	OK                   = 200
	Created              = 201
	Accepted             = 202
	NonAuthoritativeInfo = 203
	NoContent            = 204
	ResetContent         = 205
	PartialContent       = 206
	MultiStatus          = 207
	AlreadyReported      = 208
	IMUsed               = 226

	// 3xx Redirection
	MultipleChoices   = 300
	MovedPermanently  = 301
	Found             = 302
	SeeOther          = 303
	NotModified       = 304
	UseProxy          = 305
	SwitchProxy       = 306
	TemporaryRedirect = 307
	PermanentRedirect = 308

	// 4xx Client Error
	BadRequest                   = 400
	Unauthorized                 = 401
	PaymentRequired              = 402
	Forbidden                    = 403
	NotFound                     = 404
	MethodNotAllowed             = 405
	NotAcceptable                = 406
	ProxyAuthRequired            = 407
	RequestTimeout               = 408
	Conflict                     = 409
	Gone                         = 410
	LengthRequired               = 411
	PreconditionFailed           = 412
	RequestEntityTooLarge        = 413
	RequestURITooLong            = 414
	UnsupportedMediaType         = 415
	RequestedRangeNotSatisfiable = 416
	ExpectationFailed            = 417
	Teapot                       = 418
	MisdirectedRequest           = 421
	UnprocessableEntity          = 422
	Locked                       = 423
	FailedDependency             = 424
	TooEarly                     = 425
	UpgradeRequired              = 426
	PreconditionRequired         = 428
	TooManyRequests              = 429
	RequestHeaderFieldsTooLarge  = 431
	UnavailableForLegalReasons   = 451

	// 5xx Server Error
	InternalServerError           = 500
	NotImplemented                = 501
	BadGateway                    = 502
	ServiceUnavailable            = 503
	GatewayTimeout                = 504
	HTTPVersionNotSupported       = 505
	VariantAlsoNegotiates         = 506
	InsufficientStorage           = 507
	LoopDetected                  = 508
	NotExtended                   = 510
	NetworkAuthenticationRequired = 511
)

// StatusText returns a text for the HTTP status code
func StatusText(code int) string {
	switch code {
	case Continue:
		return "Continue"
	case SwitchingProtocols:
		return "Switching Protocols"
	case Processing:
		return "Processing"
	case EarlyHints:
		return "Early Hints"
	case OK:
		return "OK"
	case Created:
		return "Created"
	case Accepted:
		return "Accepted"
	case NonAuthoritativeInfo:
		return "Non-Authoritative Information"
	case NoContent:
		return "No Content"
	case ResetContent:
		return "Reset Content"
	case PartialContent:
		return "Partial Content"
	case MultiStatus:
		return "Multi-Status"
	case AlreadyReported:
		return "Already Reported"
	case IMUsed:
		return "IM Used"
	case MultipleChoices:
		return "Multiple Choices"
	case MovedPermanently:
		return "Moved Permanently"
	case Found:
		return "Found"
	case SeeOther:
		return "See Other"
	case NotModified:
		return "Not Modified"
	case UseProxy:
		return "Use Proxy"
	case SwitchProxy:
		return "Switch Proxy"
	case TemporaryRedirect:
		return "Temporary Redirect"
	case PermanentRedirect:
		return "Permanent Redirect"
	case BadRequest:
		return "Bad Request"
	case Unauthorized:
		return "Unauthorized"
	case PaymentRequired:
		return "Payment Required"
	case Forbidden:
		return "Forbidden"
	case NotFound:
		return "Not Found"
	case MethodNotAllowed:
		return "Method Not Allowed"
	case NotAcceptable:
		return "Not Acceptable"
	case ProxyAuthRequired:
		return "Proxy Authentication Required"
	case RequestTimeout:
		return "Request Timeout"
	case Conflict:
		return "Conflict"
	case Gone:
		return "Gone"
	case LengthRequired:
		return "Length Required"
	case PreconditionFailed:
		return "Precondition Failed"
	case RequestEntityTooLarge:
		return "Request Entity Too Large"
	case RequestURITooLong:
		return "Request URI Too Long"
	case UnsupportedMediaType:
		return "Unsupported Media Type"
	case RequestedRangeNotSatisfiable:
		return "Requested Range Not Satisfiable"
	case ExpectationFailed:
		return "Expectation Failed"
	case Teapot:
		return "I'm a teapot"
	case MisdirectedRequest:
		return "Misdirected Request"
	case UnprocessableEntity:
		return "Unprocessable Entity"
	case Locked:
		return "Locked"
	case FailedDependency:
		return "Failed Dependency"
	case TooEarly:
		return "Too Early"
	case UpgradeRequired:
		return "Upgrade Required"
	case PreconditionRequired:
		return "Precondition Required"
	case TooManyRequests:
		return "Too Many Requests"
	case RequestHeaderFieldsTooLarge:
		return "Request Header Fields Too Large"
	case UnavailableForLegalReasons:
		return "Unavailable For Legal Reasons"
	case InternalServerError:
		return "Internal Server Error"
	case NotImplemented:
		return "Not Implemented"
	case BadGateway:
		return "Bad Gateway"
	case ServiceUnavailable:
		return "Service Unavailable"
	case GatewayTimeout:
		return "Gateway Timeout"
	case HTTPVersionNotSupported:
		return "HTTP Version Not Supported"
	case VariantAlsoNegotiates:
		return "Variant Also Negotiates"
	case InsufficientStorage:
		return "Insufficient Storage"
	case LoopDetected:
		return "Loop Detected"
	case NotExtended:
		return "Not Extended"
	case NetworkAuthenticationRequired:
		return "Network Authentication Required"
	default:
		return "Unknown Status"
	}
}
