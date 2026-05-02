import * as React from "react";
import {
  Button,
  Flex,
  TextField,
  Text,
  Select,
  Checkbox,
  Card,
} from "@radix-ui/themes";
import { useTranslation } from "react-i18next";
import { toast } from "sonner";
import { useNodeList } from "@/contexts/NodeListContext";

type ProbeType = "auto" | "icmp" | "tcp" | "http";

const ProbeManual: React.FC = () => {
  const { t } = useTranslation();
  const { nodeList } = useNodeList();
  const [target, setTarget] = React.useState("");
  const [name, setName] = React.useState("");
  const [type, setType] = React.useState<ProbeType>("auto");
  const [interval, setInterval] = React.useState(60);
  const [selectedClients, setSelectedClients] = React.useState<Set<string>>(
    new Set(),
  );
  const [submitting, setSubmitting] = React.useState(false);

  // 默认全选
  React.useEffect(() => {
    if (nodeList && selectedClients.size === 0) {
      setSelectedClients(new Set(nodeList.map((n) => n.uuid)));
    }
  }, [nodeList]);

  const toggleAll = (on: boolean) => {
    if (!nodeList) return;
    setSelectedClients(on ? new Set(nodeList.map((n) => n.uuid)) : new Set());
  };

  const toggleOne = (uuid: string) => {
    setSelectedClients((prev) => {
      const next = new Set(prev);
      if (next.has(uuid)) next.delete(uuid);
      else next.add(uuid);
      return next;
    });
  };

  const submit = async () => {
    if (!target.trim()) {
      toast.error(t("probes.errors.target_required", { defaultValue: "请输入目标" }));
      return;
    }
    if (selectedClients.size === 0) {
      toast.error(
        t("probes.errors.no_clients", { defaultValue: "请至少选择一个节点" }),
      );
      return;
    }
    setSubmitting(true);
    try {
      const res = await fetch("/api/admin/probes/manual", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          target: target.trim(),
          name: name.trim(),
          type: type === "auto" ? "" : type,
          interval,
          clients: Array.from(selectedClients),
        }),
      });
      const text = await res.text();
      let parsed: any = null;
      try {
        parsed = text ? JSON.parse(text) : null;
      } catch {
        parsed = null;
      }
      if (!res.ok || parsed?.status === "error") {
        throw new Error(parsed?.message || text || res.statusText);
      }
      toast.success(
        parsed?.data?.created
          ? t("probes.toast.created", { defaultValue: "已添加新探测" })
          : t("probes.toast.merged", { defaultValue: "已合并到现有探测" }),
      );
      setTarget("");
      setName("");
    } catch (err: any) {
      toast.error(err?.message || "提交失败");
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <Flex direction="column" gap="4">
      <Flex direction="column" gap="3">
        <div>
          <Text as="label" size="2" weight="medium">
            {t("probes.fields.target", { defaultValue: "目标" })}
          </Text>
          <TextField.Root
            value={target}
            onChange={(e) => setTarget(e.target.value)}
            placeholder="example.com / 1.2.3.4 / 1.2.3.4:443 / https://example.com"
          />
          <Text size="1" color="gray">
            {t("probes.hints.target", {
              defaultValue:
                "host → ICMP；host:port → TCP；带 http:// 前缀 → HTTP",
            })}
          </Text>
        </div>

        <Flex gap="3" wrap="wrap">
          <div style={{ flex: 1, minWidth: 180 }}>
            <Text as="label" size="2" weight="medium">
              {t("probes.fields.name", { defaultValue: "名称（可选）" })}
            </Text>
            <TextField.Root
              value={name}
              onChange={(e) => setName(e.target.value)}
              placeholder={t("probes.placeholders.name", {
                defaultValue: "留空自动生成",
              })}
            />
          </div>
          <div style={{ width: 140 }}>
            <Text as="label" size="2" weight="medium">
              {t("probes.fields.type", { defaultValue: "协议" })}
            </Text>
            <Select.Root
              value={type}
              onValueChange={(v) => setType(v as ProbeType)}
            >
              <Select.Trigger style={{ width: "100%" }} />
              <Select.Content>
                <Select.Item value="auto">
                  {t("probes.type.auto", { defaultValue: "自动识别" })}
                </Select.Item>
                <Select.Item value="icmp">ICMP</Select.Item>
                <Select.Item value="tcp">TCP</Select.Item>
                <Select.Item value="http">HTTP</Select.Item>
              </Select.Content>
            </Select.Root>
          </div>
          <div style={{ width: 120 }}>
            <Text as="label" size="2" weight="medium">
              {t("probes.fields.interval", { defaultValue: "间隔（秒）" })}
            </Text>
            <TextField.Root
              type="number"
              min={5}
              value={interval}
              onChange={(e) =>
                setInterval(Math.max(5, parseInt(e.target.value, 10) || 60))
              }
            />
          </div>
        </Flex>

        <Card>
          <Flex justify="between" align="center" mb="2">
            <Text size="2" weight="medium">
              {t("probes.fields.clients", { defaultValue: "目标节点" })}
              <Text size="1" color="gray" ml="2">
                ({selectedClients.size}/{nodeList?.length ?? 0})
              </Text>
            </Text>
            <Flex gap="2">
              <Button
                size="1"
                variant="soft"
                onClick={() => toggleAll(true)}
                type="button"
              >
                {t("common.selectAll", { defaultValue: "全选" })}
              </Button>
              <Button
                size="1"
                variant="soft"
                color="gray"
                onClick={() => toggleAll(false)}
                type="button"
              >
                {t("common.clear", { defaultValue: "清空" })}
              </Button>
            </Flex>
          </Flex>
          <div
            style={{
              maxHeight: 200,
              overflowY: "auto",
              display: "grid",
              gridTemplateColumns: "repeat(auto-fill, minmax(180px, 1fr))",
              gap: 6,
            }}
          >
            {(nodeList ?? []).map((n) => (
              <label
                key={n.uuid}
                style={{
                  display: "flex",
                  alignItems: "center",
                  gap: 6,
                  cursor: "pointer",
                }}
              >
                <Checkbox
                  checked={selectedClients.has(n.uuid)}
                  onCheckedChange={() => toggleOne(n.uuid)}
                />
                <span className="text-sm truncate">{n.name}</span>
              </label>
            ))}
          </div>
        </Card>

        <Flex justify="end">
          <Button onClick={submit} disabled={submitting}>
            {submitting
              ? t("common.submitting", { defaultValue: "提交中..." })
              : t("probes.actions.add", { defaultValue: "添加监控" })}
          </Button>
        </Flex>
      </Flex>
    </Flex>
  );
};

export default ProbeManual;
