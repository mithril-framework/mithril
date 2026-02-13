// Package utils provides common utilities for hashing, encryption, encoding,
// string manipulation, validation, time, pagination, sorting, file operations,
// and CLI helpers.
//
// Hashing: HashPassword, CheckPasswordHash (bcrypt); HashPasswordArgon2,
// CheckPasswordArgon2 (Argon2); SHA256, SHA512, HMACSHA256, HMACSHA512;
// MD5File, SHA1File, SHA256File for files.
//
// Encryption: EncryptAES256GCM, DecryptAES256GCM (AES-256-GCM, 32-byte key).
//
// Encoding: EncodeBase64, DecodeBase64; EncodeBase64URL, DecodeBase64URL;
// EncodeBase64URLSafe, DecodeBase64URLSafe.
//
// String: GenerateRandomString, GenerateRandomBytes, Slugify, Truncate,
// TruncateWords, Capitalize, TitleCase, SnakeCase, CamelCase, PascalCase,
// RemoveDuplicates, IsEmpty, IsNotEmpty, GenerateUUID, GenerateShortUUID.
//
// Validation: IsValidEmail, IsValidPhone, IsValidURL, IsStrongPassword,
// IsValidUsername, IsValidSlug.
//
// Time: ParseDuration, FormatDuration, GetTimezoneOffset, FormatTime, TimeAgo,
// StartOfDay, EndOfDay, StartOfWeek, EndOfWeek, StartOfMonth, EndOfMonth.
//
// Pagination: PaginationMeta, PaginationLinks, PaginationResponse,
// NewPagination, GeneratePaginationLinks, ParsePaginationParams (Fiber).
//
// Sorting: SortField, ParseSortParams, SortSlice (reflect-based).
//
// File: GetFileExtension, GetFileSize, FileExists, EnsureDir, CopyFile,
// GetMIMEType, FormatFileSize.
//
// CLI: PrintSuccess, PrintError, PrintWarning, PrintInfo, AskInput,
// AskConfirmation, ShowProgress.
package utils
