import { useState } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { listSettings, createSetting, updateSetting, deleteSetting } from "@/api/settings";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { QueryErrorCard } from "@/components/ui/query-error-card";
import { describeApiError } from "@/lib/http";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Switch } from "@/components/ui/switch";
import { Textarea } from "@/components/ui/textarea";
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { SettingItem } from "@/types/settings";

type SettingValueType = "string" | "number" | "boolean" | "json" | "text";

const VALUE_TYPES: Array<{ value: SettingValueType; label: string }> = [
  { value: "string", label: "String" },
  { value: "number", label: "Number" },
  { value: "boolean", label: "Boolean" },
  { value: "json", label: "JSON" },
  { value: "text", label: "Text" }
];

export function SettingsPage() {
  const queryClient = useQueryClient();
  const [editingSetting, setEditingSetting] = useState<SettingItem | null>(null);
  const [isCreateDialogOpen, setIsCreateDialogOpen] = useState(false);
  const [isEditDialogOpen, setIsEditDialogOpen] = useState(false);
  const [isDeleteDialogOpen, setIsDeleteDialogOpen] = useState(false);

  const settingsQuery = useQuery({
    queryKey: ["settings"],
    queryFn: listSettings
  });

  const createMutation = useMutation({
    mutationFn: createSetting,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["settings"] });
      setIsCreateDialogOpen(false);
    }
  });

  const updateMutation = useMutation({
    mutationFn: ({ key, data }: { key: string; data: any }) => updateSetting(key, data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["settings"] });
      setIsEditDialogOpen(false);
    }
  });

  const deleteMutation = useMutation({
    mutationFn: (key: string) => deleteSetting(key),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["settings"] });
      setIsDeleteDialogOpen(false);
    }
  });

  const loadError = settingsQuery.isError
    ? describeApiError(settingsQuery.error, "Failed to load settings.")
    : null;

  function handleCreate() {
    setEditingSetting({
      key: "",
      value: "",
      value_type: "string",
      is_encrypted: false,
      description: "",
      created_at: "",
      updated_at: ""
    });
    setIsCreateDialogOpen(true);
  }

  function handleEdit(setting: SettingItem) {
    setEditingSetting({ ...setting });
    setIsEditDialogOpen(true);
  }

  function handleDelete(key: string) {
    setEditingSetting({ key, value: "", value_type: "string", is_encrypted: false, description: "", created_at: "", updated_at: "" });
    setIsDeleteDialogOpen(true);
  }

  const isCreating = createMutation.isPending;
  const isEditing = updateMutation.isPending;
  const isDeleting = deleteMutation.isPending;

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-2xl font-semibold text-slate-900">Settings</h2>
          <p className="text-sm text-slate-500">Manage system configuration and sensitive keys.</p>
        </div>
        <Button onClick={handleCreate}>Add Setting</Button>
      </div>

      <Card>
        <CardHeader>
          <CardTitle className="text-base">Configuration</CardTitle>
        </CardHeader>
        <CardContent>
          {settingsQuery.isLoading ? (
            <p className="text-sm text-slate-600">Loading settings...</p>
          ) : settingsQuery.isError ? (
            <QueryErrorCard
              className="shadow-none"
              title="Failed to load settings"
              message={loadError?.message || "Failed to load settings."}
              hint={loadError?.hint}
              onRetry={() => settingsQuery.refetch()}
            />
          ) : (
            <div className="overflow-hidden rounded-lg border border-slate-200">
              <table className="w-full text-left text-sm">
                <thead className="bg-slate-50 text-slate-600">
                  <tr>
                    <th className="px-4 py-3">Key</th>
                    <th className="px-4 py-3">Value Type</th>
                    <th className="px-4 py-3">Encrypted</th>
                    <th className="px-4 py-3">Description</th>
                    <th className="px-4 py-3 text-right">Actions</th>
                  </tr>
                </thead>
                <tbody>
                  {(settingsQuery.data?.items || []).map((setting) => (
                    <tr className="border-t border-slate-200" key={setting.key}>
                      <td className="px-4 py-3 font-medium">{setting.key}</td>
                      <td className="px-4 py-3">
                        <span className="rounded-full bg-slate-100 px-2 py-1 text-xs text-slate-600">
                          {setting.value_type}
                        </span>
                      </td>
                      <td className="px-4 py-3">
                        {setting.is_encrypted ? (
                          <span className="flex items-center gap-1 text-emerald-600">
                            <span className="h-2 w-2 rounded-full bg-emerald-600" />
                            Yes
                          </span>
                        ) : (
                          <span className="text-slate-500">No</span>
                        )}
                      </td>
                      <td className="px-4 py-3 text-slate-500">{setting.description || "-"}</td>
                      <td className="px-4 py-3 text-right">
                        <div className="flex items-center justify-end gap-2">
                          <Button
                            size="sm"
                            variant="ghost"
                            onClick={() => handleEdit(setting)}
                          >
                            Edit
                          </Button>
                          <Button
                            size="sm"
                            variant="ghost"
                            onClick={() => handleDelete(setting.key)}
                          >
                            Delete
                          </Button>
                        </div>
                      </td>
                    </tr>
                  ))}
                  {(settingsQuery.data?.items || []).length === 0 && (
                    <tr>
                      <td className="px-4 py-8 text-center text-slate-500" colSpan={5}>
                        No settings found. Click "Add Setting" to create one.
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
              <DialogTitle>{editingSetting?.key ? "Edit Setting" : "Create Setting"}</DialogTitle>
              <DialogDescription>
                {editingSetting?.key ? "Update the configuration value." : "Add a new configuration value."}
              </DialogDescription>
            </DialogHeader>
            <div className="space-y-4 py-4">
              <div className="space-y-2">
                <Label htmlFor="key">Key</Label>
                <Input
                  id="key"
                  value={editingSetting?.key || ""}
                  onChange={(e: React.ChangeEvent<HTMLInputElement>) => setEditingSetting({ ...editingSetting!, key: e.target.value })}
                  disabled={!!editingSetting?.key}
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="value">Value</Label>
                <Textarea
                  id="value"
                  rows={4}
                  value={editingSetting?.value || ""}
                  onChange={(e: React.ChangeEvent<HTMLTextAreaElement>) => setEditingSetting({ ...editingSetting!, value: e.target.value })}
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="value_type">Value Type</Label>
                <Select
                  value={editingSetting?.value_type}
                  onValueChange={(value: string) => setEditingSetting({ ...editingSetting!, value_type: value as SettingValueType })}
                >
                  <SelectTrigger>
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    {VALUE_TYPES.map((type) => (
                      <SelectItem key={type.value} value={type.value}>
                        {type.label}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>
              <div className="space-y-2">
                <Label htmlFor="description">Description</Label>
                <Textarea
                  id="description"
                  rows={2}
                  value={editingSetting?.description || ""}
                  onChange={(e: React.ChangeEvent<HTMLTextAreaElement>) => setEditingSetting({ ...editingSetting!, description: e.target.value })}
                />
              </div>
              <div className="flex items-center justify-between space-y-0">
                <Label htmlFor="is_encrypted">Encrypted (hide value)</Label>
                <Switch
                  id="is_encrypted"
                  checked={editingSetting?.is_encrypted}
                  onCheckedChange={(checked: boolean) => setEditingSetting({ ...editingSetting!, is_encrypted: checked })}
                />
              </div>
            </div>
            <DialogFooter>
              <Button variant="secondary" onClick={() => (isCreating ? setIsCreateDialogOpen(false) : setIsEditDialogOpen(false))}>
                Cancel
              </Button>
              <Button
                onClick={() => {
                  if (editingSetting) {
                    if (editingSetting.key) {
                      updateMutation.mutate({ key: editingSetting.key, data: editingSetting });
                    } else {
                      createMutation.mutate(editingSetting);
                    }
                  }
                }}
                disabled={!editingSetting?.key || !editingSetting?.value}
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
            <DialogTitle>Delete Setting</DialogTitle>
            <DialogDescription>
              Are you sure you want to delete the setting <span className="font-mono font-bold">{editingSetting?.key}</span>?
              This action cannot be undone.
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button variant="secondary" onClick={() => setIsDeleteDialogOpen(false)}>
              Cancel
            </Button>
            <Button variant="destructive" onClick={() => editingSetting?.key && deleteMutation.mutate(editingSetting.key)}>
              {isDeleting ? "Deleting..." : "Delete"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}
