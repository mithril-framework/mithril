# Utils

Common utilities for hashing, encryption, encoding, strings, validation, time, pagination, sorting, file operations, and CLI helpers.

**Import:**

```go
import "github.com/mithril-framework/mithril/pkg/utils"
```

---

## Hashing

**Password (bcrypt)**

```go
hash, err := utils.HashPassword("secret")
// err == nil
ok := utils.CheckPasswordHash("secret", hash) // true
```

**Password (Argon2)**

```go
hash, err := utils.HashPasswordArgon2("secret")
ok := utils.CheckPasswordArgon2("secret", hash)
```

**SHA / HMAC**

```go
hex := utils.SHA256("data")
hex := utils.SHA512("data")
mac := utils.HMACSHA256("data", "key")
mac := utils.HMACSHA512("data", "key")
```

**File hashes**

```go
md5, err := utils.MD5File("/path/to/file")
sha1, err := utils.SHA1File("/path/to/file")
sha256, err := utils.SHA256File("/path/to/file")
```

---

## Encryption

AES-256-GCM; key must be exactly 32 bytes.

```go
enc, err := utils.EncryptAES256GCM("plaintext", string(key32))
dec, err := utils.DecryptAES256GCM(enc, string(key32))
```

---

## Base64

```go
encoded := utils.EncodeBase64("hello")
decoded, err := utils.DecodeBase64(encoded)

urlEnc := utils.EncodeBase64URL("data")
urlSafe := utils.EncodeBase64URLSafe("data")  // no padding
decoded, err := utils.DecodeBase64URLSafe(urlSafe)
```

---

## String

**Random**

```go
s := utils.GenerateRandomString(16)
b := utils.GenerateRandomBytes(32)
id := utils.GenerateUUID()           // UUID v4
short := utils.GenerateShortUUID()  // 12-char hex
```

**Transform**

```go
slug := utils.Slugify("Hello World!")     // "hello-world"
s := utils.Truncate("long text", 10)      // "long te..."
s := utils.TruncateWords("one two three", 2) // "one two..."
s := utils.Capitalize("hello")            // "Hello"
s := utils.TitleCase("hello world")       // "Hello World"
s := utils.SnakeCase("HelloWorld")        // "hello_world"
s := utils.CamelCase("hello_world")       // "helloWorld"
s := utils.PascalCase("hello world")      // "HelloWorld"
```

**Helpers**

```go
list := utils.RemoveDuplicates([]string{"a", "b", "a"}) // ["a", "b"]
utils.IsEmpty("  ")   // true
utils.IsNotEmpty("x") // true
```

---

## Validation

```go
utils.IsValidEmail("user@example.com")
utils.IsValidPhone("+1 234 567 8900")
utils.IsValidURL("https://example.com")
utils.IsStrongPassword("MyP@ss1")   // needs upper, lower, digit, special, len >= 8
utils.IsValidUsername("user_1")     // 3–20 chars, alphanumeric + underscore
utils.IsValidSlug("my-post-title")   // [a-z0-9-]+ or empty
```

---

## Time

```go
d, err := utils.ParseDuration("1d")    // 24h; also "3600", "1h30m"
s := utils.FormatDuration(d)            // "1.0d", "90.0m", etc.

offset, err := utils.GetTimezoneOffset("America/New_York")
s := utils.FormatTime(t, "rfc3339")    // or "rfc822", "rfc1123", "unix", "unix_milli", etc.
s := utils.TimeAgo(t)                   // "2 hours ago", "1 day ago", etc.

start := utils.StartOfDay(t)
end := utils.EndOfDay(t)
start = utils.StartOfWeek(t)   // Monday 00:00
end = utils.EndOfWeek(t)        // Sunday 23:59
start = utils.StartOfMonth(t)
end = utils.EndOfMonth(t)
```

---

## Pagination

For use with Fiber handlers:

```go
page, perPage := utils.ParsePaginationParams(c)
meta := utils.NewPagination(page, perPage, totalCount)
links := utils.GeneratePaginationLinks("/api/items", meta)

resp := utils.PaginationResponse{
    Data:       items,
    Pagination: meta,
    Links:      links,
}
c.JSON(resp)
```

---

## Sorting

Parse query sort string (e.g. `"name,-created_at"`) and sort a slice of structs by field names:

```go
fields := utils.ParseSortParams("name,-created_at")  // name asc, created_at desc
utils.SortSlice(mySlice, fields)
```

---

## File

```go
ext := utils.GetFileExtension("file.pdf")       // ".pdf"
size, err := utils.GetFileSize("/path/to/file")
ok := utils.FileExists("/path/to/file")
err := utils.EnsureDir("/path/to/dir")
err := utils.CopyFile("/src", "/dst")
mime := utils.GetMIMEType("image.png")          // "image/png"
s := utils.FormatFileSize(1536)                 // "1.5 KB"
```

---

## CLI

Colored output and prompts (for CLI tools):

```go
utils.PrintSuccess("Done")
utils.PrintError("Failed")
utils.PrintWarning("Careful")
utils.PrintInfo("Hint")

input := utils.AskInput("Name")
ok := utils.AskConfirmation("Continue?")
utils.ShowProgress(50, 100, "Processing")
```
