package dto

type ListFilesQuery struct {
	Path string `form:"path" binding:"required"`
}

type ReadTextFileRequest struct {
	Path     string `json:"path" binding:"required"`
	MaxBytes int64  `json:"max_bytes"`
	Encoding string `json:"encoding"`
}

type WriteTextFileRequest struct {
	Path              string `json:"path" binding:"required"`
	Content           string `json:"content"`
	CreateIfNotExists bool   `json:"create_if_not_exists"`
	Truncate          bool   `json:"truncate"`
	Encoding          string `json:"encoding"`
}

type CreateDirectoryRequest struct {
	Path          string `json:"path" binding:"required"`
	CreateParents bool   `json:"create_parents"`
}

type DeleteFileRequest struct {
	Path      string `json:"path" binding:"required"`
	Recursive bool   `json:"recursive"`
}

type RenameFileRequest struct {
	SourcePath string `json:"source_path" binding:"required"`
	TargetPath string `json:"target_path" binding:"required"`
}

type DownloadFileQuery struct {
	Path   string `form:"path" binding:"required"`
	Offset uint64 `form:"offset"`
	Limit  uint64 `form:"limit"`
}

type DownloadFileResult struct {
	Path            string `json:"path"`
	StartOffset     uint64 `json:"start_offset"`
	EndOffset       uint64 `json:"end_offset"`
	TotalSize       uint64 `json:"total_size"`
	DownloadedBytes uint64 `json:"downloaded_bytes"`
}

type UploadFileRequest struct {
	Path   string `form:"path" binding:"required"`
	Offset uint64 `form:"offset"`
}

type UploadFileResult struct {
	Path          string `json:"path"`
	UploadedBytes uint64 `json:"uploaded_bytes"`
	TotalSize     uint64 `json:"total_size"`
}

type FileEntry struct {
	Name       string `json:"name"`
	Path       string `json:"path"`
	IsDir      bool   `json:"is_dir"`
	Size       uint64 `json:"size"`
	ModifiedAt int64  `json:"modified_at_unix"`
}

type ListFilesResult struct {
	CurrentPath string      `json:"current_path"`
	Entries     []FileEntry `json:"entries"`
}

type ReadTextFileResult struct {
	Path      string `json:"path"`
	Content   string `json:"content"`
	Size      uint64 `json:"size"`
	Truncated bool   `json:"truncated"`
	Encoding  string `json:"encoding"`
}

type WriteTextFileResult struct {
	Path         string `json:"path"`
	WrittenBytes uint64 `json:"written_bytes"`
}

type CreateDirectoryResult struct {
	Path string `json:"path"`
}

type DeleteFileResult struct {
	Path string `json:"path"`
}

type RenameFileResult struct {
	SourcePath   string `json:"source_path"`
	TargetPath   string `json:"target_path"`
	WrittenBytes uint64 `json:"written_bytes"`
}
