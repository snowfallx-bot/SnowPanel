import { ChangeEvent, FormEvent, useMemo, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import {
  createDirectory,
  deleteFile,
  listFiles,
  readTextFile,
  renameFile,
  writeTextFile
} from "@/api/files";
import { FileEditorPanel } from "@/components/files/FileEditorPanel";
import { FilePathBar } from "@/components/files/FilePathBar";
import { FileTable } from "@/components/files/FileTable";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
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

function decodeUtf8OrThrow(bytes: ArrayBuffer) {
  const decoder = new TextDecoder("utf-8", { fatal: true });
  return decoder.decode(bytes);
}

export function FilesPage() {
  const queryClient = useQueryClient();
  const [path, setPath] = useState("/tmp");
  const [mkdirName, setMkdirName] = useState("");
  const [selectedPath, setSelectedPath] = useState("");
  const [selectedContent, setSelectedContent] = useState("");
  const [selectedTruncated, setSelectedTruncated] = useState(false);
  const [selectedBinary, setSelectedBinary] = useState(false);
  const [readMaxBytes, setReadMaxBytes] = useState(1024 * 1024);
  const [feedback, setFeedback] = useState("");

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
      const message = error instanceof Error ? error.message : "Failed to read file";
      setSelectedPath(variables.path);
      setSelectedContent("");
      setSelectedTruncated(false);

      if (message.toLowerCase().includes("text file required")) {
        setSelectedBinary(true);
        setFeedback("This file is binary/non UTF-8 and cannot be previewed as text.");
        return;
      }

      setSelectedBinary(false);
      setFeedback(message);
    }
  });

  const writeMutation = useMutation({
    mutationFn: writeTextFile,
    onError(error) {
      setFeedback(error instanceof Error ? error.message : "Failed to write file");
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
    },
    onError(error) {
      setFeedback(error instanceof Error ? error.message : "Failed to rename path");
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
      setFeedback(error instanceof Error ? error.message : "Failed to create directory");
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
      setFeedback(error instanceof Error ? error.message : "Failed to delete path");
    }
  });

  const message = useMemo(() => {
    if (listQuery.isError) {
      return listQuery.error instanceof Error ? listQuery.error.message : "Failed to list files";
    }
    return feedback;
  }, [feedback, listQuery.error, listQuery.isError]);

  const currentPath = listQuery.data?.current_path || path;
  const canLoadMorePreview = selectedTruncated && readMaxBytes < 8 * 1024 * 1024;
  const canDownload = !selectedBinary && !!selectedPath && !selectedTruncated;

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

  async function handleOpen(entry: FileEntry) {
    if (entry.is_dir) {
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
  }

  async function handleRename(entry: FileEntry) {
    if (entry.is_dir) {
      setFeedback("Directory rename is not supported yet.");
      return;
    }

    const nextName = window.prompt("Rename file to:", entry.name);
    if (!nextName) {
      return;
    }
    const normalizedName = nextName.trim();
    if (!normalizedName || normalizedName === entry.name) {
      return;
    }
    if (normalizedName.includes("/") || normalizedName.includes("\\")) {
      setFeedback("Rename target must be a file name, not a path.");
      return;
    }

    const targetPath = joinPath(parentPath(entry.path), normalizedName);
    await renameMutation.mutateAsync({
      source_path: entry.path,
      target_path: targetPath
    });
  }

  function handleDownload() {
    if (!canDownload) {
      return;
    }
    const name = fileNameFromPath(selectedPath) || "download.txt";
    const blob = new Blob([selectedContent], { type: "text/plain;charset=utf-8" });
    const url = URL.createObjectURL(blob);
    const anchor = document.createElement("a");
    anchor.href = url;
    anchor.download = name;
    anchor.click();
    URL.revokeObjectURL(url);
  }

  async function handleUpload(event: ChangeEvent<HTMLInputElement>) {
    const file = event.target.files?.[0];
    event.target.value = "";
    if (!file) {
      return;
    }

    try {
      const content = decodeUtf8OrThrow(await file.arrayBuffer());
      const targetPath = joinPath(currentPath, file.name);
      await writeMutation.mutateAsync({
        path: targetPath,
        content,
        create_if_not_exists: true,
        truncate: true,
        encoding: "utf-8"
      });
      setFeedback(`Uploaded text file to ${targetPath}`);
      queryClient.invalidateQueries({ queryKey: ["files", path] });
    } catch (error) {
      if (error instanceof TypeError) {
        setFeedback("Upload currently supports UTF-8 text files only.");
        return;
      }
      setFeedback(error instanceof Error ? error.message : "Upload failed");
    }
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
          Browse, upload, rename, download and edit UTF-8 text files via the Rust core-agent.
        </p>
      </div>

      <FilePathBar
        onGoUp={() => setPath(parentPath(currentPath))}
        onNavigate={(target) => setPath(target)}
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
                  <input className="hidden" onChange={handleUpload} type="file" />
                  Upload UTF-8 Text File
                </label>
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

          {listQuery.isLoading ? (
            <Card>
              <CardHeader>
                <CardTitle className="text-base">Loading directory...</CardTitle>
              </CardHeader>
            </Card>
          ) : (
            <FileTable
              entries={listQuery.data?.entries || []}
              onDelete={handleDelete}
              onOpen={handleOpen}
              onRename={handleRename}
            />
          )}
        </div>

        <div className="space-y-3">
          <FileEditorPanel
            binary={selectedBinary}
            canDownload={canDownload}
            content={selectedContent}
            loading={readMutation.isPending}
            onDownload={handleDownload}
            onSave={handleSave}
            path={selectedPath || "No file selected"}
            truncated={selectedTruncated}
          />

          {canLoadMorePreview && (
            <Button
              onClick={() => {
                const nextOption = readLimitOptions.find((option) => option.value > readMaxBytes);
                if (nextOption) {
                  setPreviewLimit(nextOption.value);
                }
              }}
              size="sm"
              variant="ghost"
            >
              Increase Preview Limit
            </Button>
          )}
        </div>
      </div>

      {message && (
        <p className="rounded-md border border-slate-200 bg-white px-3 py-2 text-sm text-slate-700">{message}</p>
      )}
    </div>
  );
}
