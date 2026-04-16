import { useState } from "react"
import { useTranslation } from "react-i18next"
import { useQuery } from "@tanstack/react-query"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"
import { Badge } from "@/components/ui/badge"
import { Plus, Trash2, User, Building2, Briefcase, UserCheck, Users } from "lucide-react"
import { api } from "@/lib/api"
import type { Participant } from "../types"

interface ParticipantPickerProps {
  participants: Participant[]
  onChange: (participants: Participant[]) => void
}

interface UserItem {
  id: number
  username: string
  email: string
  avatar: string
}

interface PositionItem {
  id: number
  name: string
  code: string
}

interface DeptTreeNode {
  id: number
  name: string
  code: string
  children?: DeptTreeNode[]
}

const PARTICIPANT_TYPES = [
  { value: "user", icon: User, label: "workflow.participant.user" },
  { value: "position", icon: Briefcase, label: "workflow.participant.position" },
  { value: "department", icon: Building2, label: "workflow.participant.department" },
  { value: "position_department", icon: Users, label: "workflow.participant.positionDepartment" },
  { value: "requester_manager", icon: UserCheck, label: "workflow.participant.requesterManager" },
] as const

function formatParticipantLabel(p: Participant): string {
  if (p.type === "position_department") {
    const parts = [p.department_code, p.position_code].filter(Boolean)
    if (parts.length > 0) return parts.join(" / ")
  }
  return p.name ?? p.value ?? p.type
}

export function ParticipantPicker({ participants, onChange }: ParticipantPickerProps) {
  const { t } = useTranslation("itsm")
  const [addingType, setAddingType] = useState<string>("")
  const [userKeyword, setUserKeyword] = useState("")
  const [pdDeptCode, setPdDeptCode] = useState("")

  const { data: users } = useQuery({
    queryKey: ["users-search", userKeyword],
    queryFn: () => api.get<{ items: UserItem[] }>(`/api/v1/users?page=1&pageSize=20&keyword=${encodeURIComponent(userKeyword)}`).then((r) => r.items),
    enabled: addingType === "user" && userKeyword.length > 0,
    staleTime: 30_000,
  })

  const { data: positions } = useQuery({
    queryKey: ["org-positions"],
    queryFn: () => api.get<{ items: PositionItem[] }>("/api/v1/org/positions?pageSize=0").then((r) => r.items),
    enabled: addingType === "position" || addingType === "position_department",
    staleTime: 60_000,
  })

  const { data: departments } = useQuery({
    queryKey: ["org-departments-tree"],
    queryFn: () => api.get<{ items: DeptTreeNode[] }>("/api/v1/org/departments/tree").then((r) => r.items),
    enabled: addingType === "department" || addingType === "position_department",
    staleTime: 60_000,
  })

  function addParticipant(p: Participant) {
    onChange([...participants, p])
    setAddingType("")
    setUserKeyword("")
    setPdDeptCode("")
  }

  function removeParticipant(index: number) {
    onChange(participants.filter((_, i) => i !== index))
  }

  function handleTypeSelect(type: string) {
    if (type === "requester_manager") {
      addParticipant({ type: "requester_manager", name: t("workflow.participant.requesterManager") })
      return
    }
    if (type === "position_department") {
      setAddingType("position_department")
      return
    }
    setAddingType(type)
  }

  const flatDepts = departments ? flattenDeptTree(departments) : []

  return (
    <div className="space-y-2">
      <Label className="text-xs">{t("workflow.prop.participants")}</Label>

      {participants.length > 0 && (
        <div className="space-y-1">
          {participants.map((p, i) => (
            <div key={i} className="flex items-center justify-between rounded border px-2 py-1">
              <div className="flex items-center gap-1.5">
                <Badge variant="outline" className="text-[10px]">{p.type}</Badge>
                <span className="text-xs">{formatParticipantLabel(p)}</span>
              </div>
              <Button variant="ghost" size="icon" className="h-5 w-5" onClick={() => removeParticipant(i)}>
                <Trash2 size={12} />
              </Button>
            </div>
          ))}
        </div>
      )}

      {!addingType ? (
        <Select onValueChange={handleTypeSelect}>
          <SelectTrigger className="h-8 text-xs">
            <div className="flex items-center gap-1.5">
              <Plus size={12} />
              <SelectValue placeholder={t("workflow.participant.add")} />
            </div>
          </SelectTrigger>
          <SelectContent>
            {PARTICIPANT_TYPES.map((pt) => (
              <SelectItem key={pt.value} value={pt.value}>
                <div className="flex items-center gap-1.5">
                  <pt.icon size={12} />
                  <span>{t(pt.label)}</span>
                </div>
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      ) : addingType === "user" ? (
        <div className="space-y-1">
          <Input
            value={userKeyword}
            onChange={(e) => setUserKeyword(e.target.value)}
            placeholder={t("workflow.participant.searchUser")}
            className="h-7 text-xs"
            autoFocus
          />
          {users && users.length > 0 && (
            <div className="max-h-32 overflow-y-auto rounded border">
              {users.map((u) => (
                <button
                  key={u.id}
                  type="button"
                  className="flex w-full items-center gap-2 px-2 py-1 text-xs hover:bg-muted"
                  onClick={() => addParticipant({ type: "user", id: u.id, name: u.username, value: String(u.id) })}
                >
                  <span>{u.username}</span>
                  <span className="text-muted-foreground">{u.email}</span>
                </button>
              ))}
            </div>
          )}
          <Button variant="ghost" size="sm" className="h-6 text-xs" onClick={() => { setAddingType(""); setUserKeyword("") }}>
            {t("common.cancel")}
          </Button>
        </div>
      ) : addingType === "position" ? (
        <div className="space-y-1">
          <div className="max-h-32 overflow-y-auto rounded border">
            {(positions ?? []).map((p) => (
              <button
                key={p.id}
                type="button"
                className="flex w-full items-center gap-2 px-2 py-1 text-xs hover:bg-muted"
                onClick={() => addParticipant({ type: "position", id: p.id, name: p.name, value: p.code })}
              >
                <Briefcase size={12} />
                <span>{p.name}</span>
                <span className="text-muted-foreground">{p.code}</span>
              </button>
            ))}
          </div>
          <Button variant="ghost" size="sm" className="h-6 text-xs" onClick={() => setAddingType("")}>
            {t("common.cancel")}
          </Button>
        </div>
      ) : addingType === "department" ? (
        <div className="space-y-1">
          <div className="max-h-32 overflow-y-auto rounded border">
            {flatDepts.map((d) => (
              <button
                key={d.id}
                type="button"
                className="flex w-full items-center gap-2 px-2 py-1 text-xs hover:bg-muted"
                style={{ paddingLeft: `${(d.depth ?? 0) * 12 + 8}px` }}
                onClick={() => addParticipant({ type: "department", id: d.id, name: d.name, value: d.code })}
              >
                <Building2 size={12} />
                <span>{d.name}</span>
              </button>
            ))}
          </div>
          <Button variant="ghost" size="sm" className="h-6 text-xs" onClick={() => setAddingType("")}>
            {t("common.cancel")}
          </Button>
        </div>
      ) : addingType === "position_department" ? (
        <div className="space-y-1">
          {!pdDeptCode ? (
            <>
              <Label className="text-[11px] text-muted-foreground">{t("workflow.participant.department")}</Label>
              <div className="max-h-32 overflow-y-auto rounded border">
                {flatDepts.map((d) => (
                  <button
                    key={d.id}
                    type="button"
                    className="flex w-full items-center gap-2 px-2 py-1 text-xs hover:bg-muted"
                    style={{ paddingLeft: `${(d.depth ?? 0) * 12 + 8}px` }}
                    onClick={() => setPdDeptCode(d.code)}
                  >
                    <Building2 size={12} />
                    <span>{d.name}</span>
                    <span className="text-muted-foreground">{d.code}</span>
                  </button>
                ))}
              </div>
            </>
          ) : (
            <>
              <Label className="text-[11px] text-muted-foreground">{t("workflow.participant.position")}</Label>
              <div className="max-h-32 overflow-y-auto rounded border">
                {(positions ?? []).map((p) => (
                  <button
                    key={p.id}
                    type="button"
                    className="flex w-full items-center gap-2 px-2 py-1 text-xs hover:bg-muted"
                    onClick={() => addParticipant({ type: "position_department", department_code: pdDeptCode, position_code: p.code })}
                  >
                    <Briefcase size={12} />
                    <span>{p.name}</span>
                    <span className="text-muted-foreground">{p.code}</span>
                  </button>
                ))}
              </div>
            </>
          )}
          <Button variant="ghost" size="sm" className="h-6 text-xs" onClick={() => { setAddingType(""); setPdDeptCode("") }}>
            {t("common.cancel")}
          </Button>
        </div>
      ) : null}
    </div>
  )
}

function flattenDeptTree(nodes: DeptTreeNode[], depth = 0): Array<DeptTreeNode & { depth: number }> {
  const result: Array<DeptTreeNode & { depth: number }> = []
  for (const n of nodes) {
    result.push({ ...n, depth })
    if (n.children) {
      result.push(...flattenDeptTree(n.children, depth + 1))
    }
  }
  return result
}
