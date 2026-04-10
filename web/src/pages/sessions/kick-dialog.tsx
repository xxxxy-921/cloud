import { useMutation, useQueryClient } from "@tanstack/react-query"
import { api } from "@/lib/api"
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog"

interface KickDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  sessionId: number | null
  username: string
}

export function KickDialog({ open, onOpenChange, sessionId, username }: KickDialogProps) {
  const queryClient = useQueryClient()

  const mutation = useMutation({
    mutationFn: (id: number) => api.delete(`/api/v1/sessions/${id}`),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["sessions"] })
      onOpenChange(false)
    },
  })

  return (
    <AlertDialog open={open} onOpenChange={onOpenChange}>
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>踢出会话</AlertDialogTitle>
          <AlertDialogDescription>
            确定要强制下线用户 &ldquo;{username}&rdquo; 的这个会话吗？该用户将立即被登出。
          </AlertDialogDescription>
        </AlertDialogHeader>
        <AlertDialogFooter>
          <AlertDialogCancel disabled={mutation.isPending}>取消</AlertDialogCancel>
          <AlertDialogAction
            onClick={() => sessionId && mutation.mutate(sessionId)}
            disabled={mutation.isPending}
          >
            {mutation.isPending ? "处理中..." : "确认踢出"}
          </AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  )
}
