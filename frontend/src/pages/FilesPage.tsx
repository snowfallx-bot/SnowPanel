import { ChangeEvent, DragEvent, FormEvent, useMemo, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import {
  createDirectory,
  deleteFile,
  downloadFile,
  listFiles,
  readTextFile,
  renameFile,
  uploadFileWithRetry,
  writeTextFile
} from "@/api/files";
import { FileEditorPanel } from "@/components/files/FileEditorPanel";
import { FilePathBar } from "@/components/files/FilePathBar";
import { FileTable } from "@/components/files/FileTable";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { QueryErrorCard } from "@/components/ui/query-error-card";
import { ApiError } from "@/lib/http";
import { FileEntry } from "@/types/file";

const readLimitOptions = [
  { label: "256 KB", value: 256 * 1024 },
  { label: "1 MB", value: 1024 * 1024 },
  { label: "4 MB", value: 4 * 1024 * 1024 },
  { label: "8 MB", value: 8 * 1024 * 1024 }
];

function parentPath(path: string) {
  if (!path || path === "/") {
    return "/";
  }
  const normalized = path.replace(/\\/g, "/").replace(/\/+$/, "");
  const index = normalized.lastIndexOf("/");
  if (index <= 0) {
    return "/";
  }
  return normalized.slice(0, index);
}

function joinPath(base: string, name: string) {
  return `${base.replace(/\/$/, "")}/${name}`.replace("//", "/");
}

function fileNameFromPath(path: string) {
  const normalized = path.replace(/\\/g, "/").replace(/\/+$/, "");
  const index = normalized.lastIndexOf("/");
  if (index < 0) {
    return normalized;
  }
  return normalized.slice(index + 1);
}

function describeFileApiError(error: unknown, fallback: string) {
  if (error instanceof ApiError) {
    switch (error.code) {
      case 3001:
        return "core-agent is unavailable. Please check backend and agent connectivity.";
      case 4000:
        return "Invalid request. Please verify file path and input.";
      case 4001:
        return "Path is outside the allowed safe roots.";
      case 4002:
        return "Target path was not found.";
      case 4003:
        return "This file is binary or non UTF-8 text.";
      case 4004:
        return "File is too large for the current operation or preview limit.";
      case 4005:
        return "I/O error while accessing the file. Check path permissions and file locks.";
      case 4006:
        return "Unsupported encoding. Use UTF-8.";
      case 4007:
        return "Dangerous path is blocked by security policy.";
      default:
        return error.message || fallback;
    }
  }

  if (error instanceof Error) {
    return error.message || fallback;
  }
  return fallback;
}

function formatProgress(current: number, total: number | null) {
  if (total !== null && total > 0) {
    const percent = Math.min(100, Math.round((current / total) * 100));
    return `${percent}% (${current}/${total} bytes)`;
  }
  return `${current} bytes`;
}

function triggerBrowserDownload(blob: Blob, fileName: string) {
  const url = URL.createObjectURL(blob);
  const anchor = document.createElement("a");
  anchor.href = url;
  anchor.download = fileName;
  anchor.click();
  URL.revokeObjectURL(url);
}

export function FilesPage() {
  const queryClient = useQueryClient();
  const [path, setPath] = useState("/tmp");
  const [mkdirName, setMkdirName] = useState("");
  const [selectedPath, setSelectedPath] = useState("");
  const [selectedContent, setSelectedContent] = useState("");
  const [selectedTruncated, setSelectedTruncated] = useState(false);
  const [selectedBinary, setSelectedBinary] = useState(false);
  const [selectedEntries, setSelectedEntries] = useState<string[]>([]);
  const [readMaxBytes, setReadMaxBytes] = useState(1024 * 1024);
  const [feedback, setFeedback] = useState("");
  const [uploadProgressText, setUploadProgressText] = useState("");
  const [downloadProgressText, setDownloadProgressText] = useState("");
  const [downloading, setDownloading] = useState(false);
  const [uploading, setUploading] = useState(false);
  const [dragging, setDragging] = useState(false);

  const listQuery = useQuery({
    queryKey: ["files", path],
    queryFn: () => listFiles(path)
  });

  const readMutation = useMutation({
    mutationFn: readTextFile,
    onSuccess(data) {
      setSelectedPath(data.path);
      setSelectedContent(data.content);
      setSelectedTruncated(data.truncated);
      setSelectedBinary(false);
      setFeedback("");
    },
    onError(error, variables) {
      const message = describeFileApiError(error, "Failed to read file");
      setSelectedPath(variables.path);
      setSelectedContent("");
      setSelectedTruncated(false);
      if (error instanceof ApiError && error.code === 4003) {
        setSelectedBinary(true);
        setFeedback(message);
        return;
      }

      setSelectedBinary(false);
      setFeedback(message);
    }
  });

  const writeMutation = useMutation({
    mutationFn: writeTextFile,
    onError(error) {
      setFeedback(describeFileApiError(error, "Failed to write file"));
    }
  });

  const renameMutation = useMutation({
    mutationFn: renameFile,
    onSuccess(result) {
      setFeedback(`Renamed: ${result.source_path} -> ${result.target_path}`);
      queryClient.invalidateQueries({ queryKey: ["files", path] });
      if (selectedPath === result.source_path) {
        setSelectedPath(result.target_path);
      }
      setSelectedEntries((current) => current.filter((item) => item !== result.source_path));
    },
    onError(error) {
      setFeedback(describeFileApiError(error, "Failed to rename path"));
    }
  });

  const mkdirMutation = useMutation({
    mutationFn: createDirectory,
    onSuccess() {
      setMkdirName("");
      setFeedback("Directory created.");
      queryClient.invalidateQueries({ queryKey: ["files", path] });
    },
    onError(error) {
      setFeedback(describeFileApiError(error, "Failed to create directory"));
    }
  });

  const deleteMutation = useMutation({
    mutationFn: deleteFile,
    onSuccess() {
      setFeedback("Path deleted.");
      queryClient.invalidateQueries({ queryKey: ["files", path] });
      if (selectedPath) {
        setSelectedPath("");
        setSelectedContent("");
        setSelectedTruncated(false);
        setSelectedBinary(false);
      }
    },
    onError(error) {
      setFeedback(describeFileApiError(error, "Failed to delete path"));
    }
  });

  const message = useMemo(() => {
    if (uploadProgressText) {
      return uploadProgressText;
    }
    return feedback;
  }, [feedback, uploadProgressText]);

  const currentPath = listQuery.data?.current_path || path;
  const currentEntries = listQuery.data?.entries || [];
  const canLoadMorePreview = selectedTruncated && readMaxBytes < 8 * 1024 * 1024;
  const canDownload = !!selectedPath;
  const hasBulkSelection = selectedEntries.length > 0;

  function setPreviewLimit(nextMaxBytes: number) {
    setReadMaxBytes(nextMaxBytes);
    if (!selectedPath || selectedBinary) {
      return;
    }
    readMutation.mutate({
      path: selectedPath,
      max_bytes: nextMaxBytes,
      encoding: "utf-8"
    });
  }

  function toggleSelect(entry: FileEntry) {
    setSelectedEntries((current) =>
      current.includes(entry.path) ? current.filter((item) => item !== entry.path) : [...current, entry.path]
    );
  }

  function toggleSelectAll() {
    setSelectedEntries((current) => {
      if (currentEntries.length > 0 && currentEntries.every((entry) => current.includes(entry.path))) {
        return [];
      }
      return currentEntries.map((entry) => entry.path);
    });
  }

  async function handleOpen(entry: FileEntry) {
    if (entry.is_dir) {
      setSelectedEntries([]);
      setPath(entry.path);
      return;
    }
    await readMutation.mutateAsync({
      path: entry.path,
      max_bytes: readMaxBytes,
      encoding: "utf-8"
    });
  }

  async function handleSave(content: string) {
    if (!selectedPath) {
      throw new Error("No file selected");
    }
    await writeMutation.mutateAsync({
      path: selectedPath,
      content,
      create_if_not_exists: false,
      truncate: true,
      encoding: "utf-8"
    });
    setSelectedContent(content);
    setFeedback("File saved.");
    queryClient.invalidateQueries({ queryKey: ["files", path] });
  }

  async function handleDelete(entry: FileEntry) {
    const ok = window.confirm(
      `Delete ${entry.is_dir ? "directory" : "file"} "${entry.path}"? This action cannot be undone.`
    );
    if (!ok) {
      return;
    }

    await deleteMutation.mutateAsync({
      path: entry.path,
      recursive: entry.is_dir
    });
    setSelectedEntries((current) => current.filter((item) => item !== entry.path));
  }

  async function handleBulkDelete() {
    if (!hasBulkSelection) {
      return;
    }

    const selectedItems = currentEntries.filter((entry) => selectedEntries.includes(entry.path));
    const ok = window.confirm(`Delete ${selectedItems.length} selected item(s)? This action cannot be undone.`);
    if (!ok) {
      return;
    }

    for (const entry of selectedItems) {
      await deleteMutation.mutateAsync({
        path: entry.path,
        recursive: entry.is_dir
      });
    }

    setSelectedEntries([]);
    setFeedback(`Deleted ${selectedItems.length} item(s).`);
  }

  async function handleBulkDownload() {
    if (!hasBulkSelection || downloading) {
      return;
    }

    const selectedFiles = currentEntries.filter((entry) => selectedEntries.includes(entry.path) && !entry.is_dir);
    if (selectedFiles.length === 0) {
      setFeedback("Bulk download only supports files. Selected directories were skipped.");
      return;
    }

    setDownloading(true);
    try {
      for (let index = 0; index < selectedFiles.length; index += 1) {
        const entry = selectedFiles[index];
        setDownloadProgressText(`Downloading ${index + 1}/${selectedFiles.length}: ${entry.name}`);
        const fallbackName = fileNameFromPath(entry.path) || `download-${index + 1}.bin`;
        const { blob, fileName } = await downloadFile(entry.path, (downloadedBytes, totalBytes) => {
          setDownloadProgressText(
            `Downloading ${index + 1}/${selectedFiles.length}: ${entry.name} — ${formatProgress(downloadedBytes, totalBytes)}`
          );
        });
        triggerBrowserDownload(blob, fileName || fallbackName);
      }
      setFeedback(`Downloaded ${selectedFiles.length} file(s).`);
    } catch (error) {
      setFeedback(describeFileApiError(error, "Bulk download failed"));
    } finally {
      setDownloading(false);
      setDownloadProgressText("");
    }
  }

  async function handleRename(entry: FileEntry) {
    const nextName = window.prompt(`Rename ${entry.is_dir ? "directory" : "file"} to:`, entry.name);
    if (!nextName) {
      return;
    }
    const normalizedName = nextName.trim();
    if (!normalizedName || normalizedName === entry.name) {
      return;
    }
    if (normalizedName.includes("/") || normalizedName.includes("\\")) {
      setFeedback("Rename target must be a single name, not a path.");
      return;
    }

    const targetPath = joinPath(parentPath(entry.path), normalizedName);
    await renameMutation.mutateAsync({
      source_path: entry.path,
      target_path: targetPath
    });
  }

  async function handleDownload() {
    if (!canDownload || downloading) {
      return;
    }
    setDownloading(true);
    setDownloadProgressText("Starting download...");
    try {
      const fallbackName = fileNameFromPath(selectedPath) || "download.bin";
      const { blob, fileName } = await downloadFile(selectedPath, (downloadedBytes, totalBytes) => {
        setDownloadProgressText(`Downloading: ${formatProgress(downloadedBytes, totalBytes)}`);
      });
      const name = fileName || fallbackName;
      triggerBrowserDownload(blob, name);
      setFeedback(`Downloaded ${name}`);
    } catch (error) {
      setFeedback(describeFileApiError(error, "Download failed"));
    } finally {
      setDownloading(false);
      setDownloadProgressText("");
    }
  }

  async function uploadSingleFile(file: File) {
    if (uploading) {
      return;
    }

    setUploading(true);
    setUploadProgressText("Starting upload...");
    try {
      const targetPath = joinPath(currentPath, file.name);
      const result = await uploadFileWithRetry(file, targetPath, (uploadedBytes, totalBytes) => {
        setUploadProgressText(`Uploading: ${formatProgress(uploadedBytes, totalBytes)}`);
      });
      setFeedback(`Uploaded ${file.name} (${result.uploaded_bytes} bytes) to ${result.path}`);
      queryClient.invalidateQueries({ queryKey: ["files", path] });
    } catch (error) {
      setFeedback(describeFileApiError(error, "Upload failed"));
    } finally {
      setUploading(false);
      setUploadProgressText("");
    }
  }

  async function handleUpload(event: ChangeEvent<HTMLInputElement>) {
    const file = event.target.files?.[0];
    event.target.value = "";
    if (!file || uploading) {
      return;
    }

    await uploadSingleFile(file);
  }

  function handleDragOver(event: DragEvent<HTMLDivElement>) {
    event.preventDefault();
    if (!uploading) {
      setDragging(true);
    }
  }

  function handleDragLeave(event: DragEvent<HTMLDivElement>) {
    if (event.currentTarget.contains(event.relatedTarget as Node | null)) {
      return;
    }
    setDragging(false);
  }

  async function handleDrop(event: DragEvent<HTMLDivElement>) {
    event.preventDefault();
    setDragging(false);
    if (uploading) {
      return;
    }

    const file = event.dataTransfer.files?.[0];
    if (!file) {
      return;
    }

    await uploadSingleFile(file);
  }

  async function handleCreateDirectory(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const name = mkdirName.trim();
    if (!name) {
      return;
    }
    const nextPath = joinPath(currentPath, name);
    await mkdirMutation.mutateAsync({
      path: nextPath,
      create_parents: false
    });
  }

  return (
    <div className="space-y-4">
      <div>
        <h2 className="text-2xl font-semibold text-slate-900">Files</h2>
        <p className="text-sm text-slate-500">
          Browse, upload, rename, download and edit files via the Rust core-agent.
        </p>
      </div>

      <FilePathBar
        onGoUp={() => {
          setSelectedEntries([]);
          setPath(parentPath(currentPath));
        }}
        onNavigate={(target) => {
          setSelectedEntries([]);
          setPath(target);
        }}
        path={currentPath}
      />

      <div className="grid gap-4 xl:grid-cols-[2fr_1fr]">
        <div className="space-y-4">
          <Card>
            <CardHeader>
              <CardTitle className="text-base">Directory Actions</CardTitle>
            </CardHeader>
            <CardContent className="space-y-3">
              <form className="flex gap-2" onSubmit={handleCreateDirectory}>
                <Input
                  onChange={(event) => setMkdirName(event.target.value)}
                  placeholder="new-folder-name"
                  value={mkdirName}
                />
                <Button disabled={mkdirMutation.isPending} type="submit">
                  {mkdirMutation.isPending ? "Creating..." : "Create"}
                </Button>
              </form>

              <div className="flex items-center gap-2">
                <label className="rounded-md border border-slate-300 px-3 py-2 text-sm text-slate-700">
                  <input className="hidden" disabled={uploading} onChange={handleUpload} type="file" />
                  {uploading ? "Uploading..." : "Upload File"}
                </label>
                <Button disabled={!hasBulkSelection || downloading} onClick={handleBulkDownload} type="button" variant="ghost">
                  Download Selected
                </Button>
                <Button disabled={!hasBulkSelection || deleteMutation.isPending} onClick={handleBulkDelete} type="button" variant="ghost">
                  Delete Selected
                </Button>
                <label className="text-sm text-slate-500">Preview limit</label>
                <select
                  className="rounded-md border border-slate-300 px-2 py-1 text-sm"
                  onChange={(event) => setPreviewLimit(Number(event.target.value))}
                  value={readMaxBytes}
                >
                  {readLimitOptions.map((option) => (
                    <option key={option.value} value={option.value}>
                      {option.label}
                    </option>
                  ))}
                </select>
              </div>
            </CardContent>
          </Card>

          <div
            className={`rounded-lg border-2 border-dashed p-4 transition ${dragging ? "border-panel-500 bg-panel-50" : "border-slate-300 bg-slate-50"}`}
            onDragLeave={handleDragLeave}
            onDragOver={handleDragOver}
            onDrop={handleDrop}
          >
            <p className="text-sm text-slate-600">
              {uploading
                ? "Uploading file..."
                : dragging
                  ? "Drop the file here to upload into the current directory."
                  : "Drag and drop a file here to upload it into the current directory."}
            </p>
          </div>

          {listQuery.isLoading ? (
            <Card>
              <CardHeader>
                <CardTitle className="text-base">Loading directory...</CardTitle>
              </CardHeader>
            </Card>
          ) : listQuery.isError ? (
            <QueryErrorCard
              title="Failed to load directory"
              message={describeFileApiError(listQuery.error, "Failed to list files")}
              onRetry={() => listQuery.refetch()}
            />
          ) : (
            <FileTable
              entries={currentEntries}
              selectedPaths={selectedEntries}
              onDelete={handleDelete}
              onOpen={handleOpen}
              onRename={handleRename}
              onToggleSelect={toggleSelect}
              onToggleSelectAll={toggleSelectAll}
            />
          )}
        </div>

        <div className="space-y-3">
          <FileEditorPanel
            binary={selectedBinary}
            canDownload={canDownload}
            content={selectedContent}
            downloading={downloading}
            downloadProgressText={downloadProgressText}
            loading={readMutation.isPending || writeMutation.isPending}
            onDownload={handleDownload}
            onSave={handleSave}
            path={selectedPath || "No file selected"}
            truncated={selectedTruncated}
          />

          {message ? (
            <Card>
              <CardContent className="pt-6 text-sm text-slate-600">{message}</CardContent>
            </Card>
          ) : null}
        </div>
      </div>
    </div>
  );
}
