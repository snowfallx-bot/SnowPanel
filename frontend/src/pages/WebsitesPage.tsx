import { useState } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { listWebsites, createWebsite, updateWebsite, deleteWebsite, enableWebsite, disableWebsite } from "@/api/website";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { QueryErrorCard } from "@/components/ui/query-error-card";
import { describeApiError } from "@/lib/http";
import { hostScopeKey, useHostStore } from "@/store/host-store";
import { Website, WebsiteRuntime, WebsiteStatus } from "@/types/website";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog";

const RUNTIME_OPTIONS: Array<{ value: WebsiteRuntime; label: string }> = [
  { value: "php", label: "PHP" },
  { value: "node", label: "Node.js" },
  { value: "python", label: "Python" },
  { value: "static", label: "Static" },
];

export function WebsitesPage() {
  const queryClient = useQueryClient();
  const selectedHostId = useHostStore((state) => state.selectedHostId);
  const hostScope = hostScopeKey(selectedHostId);
  const [editingWebsite, setEditingWebsite] = useState<Website | null>(null);
  const [isCreateDialogOpen, setIsCreateDialogOpen] = useState(false);
  const [isEditDialogOpen, setIsEditDialogOpen] = useState(false);
  const [isDeleteDialogOpen, setIsDeleteDialogOpen] = useState(false);

  const websitesQuery = useQuery({
    queryKey: ["websites", hostScope],
    queryFn: listWebsites,
  });

  const createMutation = useMutation({
    mutationFn: createWebsite,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["websites", hostScope] });
      setIsCreateDialogOpen(false);
    },
  });

  const updateMutation = useMutation({
    mutationFn: ({ id, data }: { id: number; data: any }) => updateWebsite(id, data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["websites", hostScope] });
      setIsEditDialogOpen(false);
    },
  });

  const deleteMutation = useMutation({
    mutationFn: (id: number) => deleteWebsite(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["websites", hostScope] });
      setIsDeleteDialogOpen(false);
    },
  });

  const enableMutation = useMutation({
    mutationFn: (id: number) => enableWebsite(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["websites", hostScope] });
    },
  });

  const disableMutation = useMutation({
    mutationFn: (id: number) => disableWebsite(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["websites", hostScope] });
    },
  });

  const loadError = websitesQuery.isError
    ? describeApiError(websitesQuery.error, "Failed to load websites.")
    : null;

  function handleCreate() {
    setEditingWebsite({
      id: 0,
      name: "",
      root_path: "",
      runtime: "php",
      status: 1,
      domains: [],
      created_at: "",
      updated_at: "",
    });
    setIsCreateDialogOpen(true);
  }

  function handleEdit(website: Website) {
    setEditingWebsite({ ...website });
    setIsEditDialogOpen(true);
  }

  function handleDelete(id: number) {
    setEditingWebsite(null);
    setIsDeleteDialogOpen(true);
  }

  const isCreating = createMutation.isPending;
  const isEditing = updateMutation.isPending;
  const isDeleting = deleteMutation.isPending;

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-2xl font-semibold text-slate-900">Websites</h2>
          <p className="text-sm text-slate-500">Manage website configurations and domains.</p>
        </div>
        <Button onClick={handleCreate}>Add Website</Button>
      </div>

      <Card>
        <CardHeader>
          <CardTitle className="text-base">Website List</CardTitle>
        </CardHeader>
        <CardContent>
          {websitesQuery.isLoading ? (
            <p className="text-sm text-slate-600">Loading websites...</p>
          ) : websitesQuery.isError ? (
            <QueryErrorCard
              className="shadow-none"
              title="Failed to load websites"
              message={loadError?.message || "Failed to load websites."}
              hint={loadError?.hint}
              onRetry={() => websitesQuery.refetch()}
            />
          ) : (
            <div className="overflow-hidden rounded-lg border border-slate-200">
              <table className="w-full text-left text-sm">
                <thead className="bg-slate-50 text-slate-600">
                  <tr>
                    <th className="px-4 py-3">Name</th>
                    <th className="px-4 py-3">Runtime</th>
                    <th className="px-4 py-3">Root Path</th>
                    <th className="px-4 py-3">Status</th>
                    <th className="px-4 py-3 text-right">Actions</th>
                  </tr>
                </thead>
                <tbody>
                  {(websitesQuery.data?.items || []).map((website) => (
                    <tr className="border-t border-slate-200" key={website.id}>
                      <td className="px-4 py-3 font-medium">{website.name}</td>
                      <td className="px-4 py-3">
                        <span className="rounded-full bg-slate-100 px-2 py-1 text-xs text-slate-600">
                          {website.runtime}
                        </span>
                      </td>
                      <td className="px-4 py-3 text-slate-600">{website.root_path}</td>
                      <td className="px-4 py-3">
                        {website.status === 1 ? (
                          <span className="flex items-center gap-1 text-emerald-600">
                            <span className="h-2 w-2 rounded-full bg-emerald-600" />
                            Enabled
                          </span>
                        ) : (
                          <span className="flex items-center gap-1 text-rose-600">
                            <span className="h-2 w-2 rounded-full bg-rose-600" />
                            Disabled
                          </span>
                        )}
                      </td>
                      <td className="px-4 py-3 text-right">
                        <div className="flex items-center justify-end gap-2">
                          <Button
                            size="sm"
                            variant="ghost"
                            onClick={() => handleEdit(website)}
                          >
                            Edit
                          </Button>
                          {website.status === 1 ? (
                            <Button
                              size="sm"
                              variant="ghost"
                              onClick={() => disableMutation.mutate(website.id)}
                            >
                              Disable
                            </Button>
                          ) : (
                            <Button
                              size="sm"
                              variant="ghost"
                              onClick={() => enableMutation.mutate(website.id)}
                            >
                              Enable
                            </Button>
                          )}
                          <Button
                            size="sm"
                            variant="ghost"
                            onClick={() => handleDelete(website.id)}
                          >
                            Delete
                          </Button>
                        </div>
                      </td>
                    </tr>
                  ))}
                  {(websitesQuery.data?.items || []).length === 0 && (
                    <tr>
                      <td className="px-4 py-8 text-center text-slate-500" colSpan={5}>
                        No websites found. Click "Add Website" to create one.
                      </td>
                    </tr>
                  )}
                </tbody>
              </table>
            </div>
          )}
        </CardContent>
      </Card>

      {/* Create/Edit Dialog */}
      {(isCreating || isEditing) && (
        <Dialog open={isCreating || isEditing} onOpenChange={(open) => !open && (isCreating ? setIsCreateDialogOpen(false) : setIsEditDialogOpen(false))}>
          <DialogContent>
            <DialogHeader>
              <DialogTitle>{editingWebsite?.id ? "Edit Website" : "Create Website"}</DialogTitle>
              <DialogDescription>
                {editingWebsite?.id ? "Update the website configuration." : "Create a new website configuration."}
              </DialogDescription>
            </DialogHeader>
            <div className="space-y-4 py-4">
              <div className="space-y-2">
                <Label htmlFor="name">Name</Label>
                <Input
                  id="name"
                  value={editingWebsite?.name || ""}
                  onChange={(e: React.ChangeEvent<HTMLInputElement>) => setEditingWebsite({ ...editingWebsite!, name: e.target.value })}
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="root_path">Root Path</Label>
                <Input
                  id="root_path"
                  value={editingWebsite?.root_path || ""}
                  onChange={(e: React.ChangeEvent<HTMLInputElement>) => setEditingWebsite({ ...editingWebsite!, root_path: e.target.value })}
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="runtime">Runtime</Label>
                <Select
                  value={editingWebsite?.runtime}
                  onValueChange={(value: WebsiteRuntime) => setEditingWebsite({ ...editingWebsite!, runtime: value })}
                >
                  <SelectTrigger>
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    {RUNTIME_OPTIONS.map((runtime) => (
                      <SelectItem key={runtime.value} value={runtime.value}>
                        {runtime.label}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>
              <div className="space-y-2">
                <Label htmlFor="domains">Domains (comma-separated)</Label>
                <Input
                  id="domains"
                  value={(editingWebsite?.domains || []).join(", ")}
                  onChange={(e: React.ChangeEvent<HTMLInputElement>) => setEditingWebsite({
                    ...editingWebsite!,
                    domains: e.target.value.split(",").map(d => d.trim()).filter(d => d.length > 0),
                  })}
                />
              </div>
            </div>
            <DialogFooter>
              <Button variant="secondary" onClick={() => (isCreating ? setIsCreateDialogOpen(false) : setIsEditDialogOpen(false))}>
                Cancel
              </Button>
              <Button
                onClick={() => {
                  if (editingWebsite) {
                    if (editingWebsite.id) {
                      updateMutation.mutate({ id: editingWebsite.id, data: editingWebsite });
                    } else {
                      createMutation.mutate(editingWebsite);
                    }
                  }
                }}
                disabled={!editingWebsite?.name || !editingWebsite?.root_path || !editingWebsite?.domains || editingWebsite.domains.length === 0}
              >
                {isCreating || isEditing ? "Saving..." : "Save"}
              </Button>
            </DialogFooter>
          </DialogContent>
        </Dialog>
      )}

      {/* Delete Confirmation Dialog */}
      <Dialog open={isDeleteDialogOpen} onOpenChange={setIsDeleteDialogOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Delete Website</DialogTitle>
            <DialogDescription>
              Are you sure you want to delete this website? All domains and configurations will be removed.
              This action cannot be undone.
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button variant="secondary" onClick={() => setIsDeleteDialogOpen(false)}>
              Cancel
            </Button>
            <Button variant="destructive" onClick={() => editingWebsite && deleteMutation.mutate(editingWebsite.id)}>
              {isDeleting ? "Deleting..." : "Delete"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}
