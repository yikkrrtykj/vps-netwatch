import * as React from "react";
import {
  Button,
  Flex,
  TextField,
  Text,
  Card,
  Badge,
  Checkbox,
  Select,
} from "@radix-ui/themes";
import { useTranslation } from "react-i18next";
import { toast } from "sonner";
import { useNodeList } from "@/contexts/NodeListContext";

type ClashTarget = {
  target: string;
  type: "icmp" | "tcp" | "http";
  host: string;
  port?: number;
  network?: string;
  rule?: string;
  process?: string;
  chains?: string[];
  count: number;
};

type CoverMode = "all" | "include" | "exclude";

const ClashHelper: React.FC = () => {
  const { t } = useTranslation();
  const { nodeList } = useNodeList();
  const [controller, setController] = React.useState("http://127.0.0.1:9090");
  const [secret, setSecret] = React.useState("");
  const [scanning, setScanning] = React.useState(false);
  const [submitting, setSubmitting] = React.useState(false);
  const [targets, setTargets] = React.useState<ClashTarget[]>([]);
  const [selected, setSelected] = React.useState<Set<string>>(new Set());
  const [interval, setInterval] = React.useState(60);
  const [cover, setCover] = React.useState<CoverMode>("all");
  const [selectedClients, setSelectedClients] = React.useState<Set<string>>(
    new Set(),
  );

  // include 默认全选，exclude 默认空
  React.useEffect(() => {
    if (!nodeList) return;
    if (cover === "include" && selectedClients.size === 0) {
      setSelectedClients(new Set(nodeList.map((n) => n.uuid)));
    }
    if (cover === "exclude") {
      setSelectedClients(new Set());
    }
  }, [cover, nodeList]);

  const coverNumber = cover === "all" ? 1 : cover === "exclude" ? 2 : 0;

  const scan = async () => {
    setScanning(true);
    setTargets([]);
    setSelected(new Set());
    try {
      const res = await fetch("/api/admin/probes/clash/discover", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          controller: controller.trim(),
          secret: secret.trim(),
          limit: 100,
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
      const list = (parsed?.data?.targets || []) as ClashTarget[];
      setTargets(list);
      if (list.length === 0) {
        toast.info(
          t("probes.clash.empty", {
            defaultValue: "Clash 当前无活跃连接",
          }),
        );
      }
    } catch (err: any) {
      toast.error(err?.message || "扫描失败");
    } finally {
      setScanning(false);
    }
  };

  const toggleSelect = (key: string) => {
    setSelected((prev) => {
      const next = new Set(prev);
      if (next.has(key)) next.delete(key);
      else next.add(key);
      return next;
    });
  };

  const toggleClient = (uuid: string) => {
    setSelectedClients((prev) => {
      const next = new Set(prev);
      if (next.has(uuid)) next.delete(uuid);
      else next.add(uuid);
      return next;
    });
  };

  const submit = async () => {
    if (selected.size === 0) {
      toast.error(
        t("probes.errors.no_targets", {
          defaultValue: "请至少选择一个目标",
        }),
      );
      return;
    }
    if (cover === "include" && selectedClients.size === 0) {
      toast.error(
        t("probes.errors.no_clients", { defaultValue: "请至少选择一个节点" }),
      );
      return;
    }
    setSubmitting(true);
    let success = 0;
    let failed = 0;
    for (const key of selected) {
      const tgt = targets.find((x) => `${x.type}|${x.target}` === key);
      if (!tgt) continue;
      try {
        const res = await fetch("/api/admin/probes/manual", {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({
            target: tgt.target,
            name: tgt.process
              ? `${tgt.process} ${tgt.target}`
              : `${tgt.type.toUpperCase()} ${tgt.target}`,
            type: tgt.type,
            interval,
            clients: Array.from(selectedClients),
            cover: coverNumber,
          }),
        });
        if (!res.ok) {
          failed++;
        } else {
          success++;
        }
      } catch {
        failed++;
      }
    }
    setSubmitting(false);
    if (success > 0) {
      toast.success(
        t("probes.clash.added", {
          defaultValue: `成功添加 ${success} 个`,
          count: success,
        }),
      );
      setSelected(new Set());
    }
    if (failed > 0) {
      toast.error(
        t("probes.clash.failed", {
          defaultValue: `${failed} 个失败`,
          count: failed,
        }),
      );
    }
  };

  return (
    <Flex direction="column" gap="4">
      <Card>
        <Flex direction="column" gap="3">
          <Flex gap="3" wrap="wrap" align="end">
            <div style={{ flex: 1, minWidth: 240 }}>
              <Text as="label" size="2" weight="medium">
                {t("probes.clash.controller", {
                  defaultValue: "Clash / mihomo controller URL",
                })}
              </Text>
              <TextField.Root
                value={controller}
                onChange={(e) => setController(e.target.value)}
                placeholder="http://127.0.0.1:9090"
              />
            </div>
            <div style={{ width: 220 }}>
              <Text as="label" size="2" weight="medium">
                {t("probes.clash.secret", { defaultValue: "Secret（可选）" })}
              </Text>
              <TextField.Root
                value={secret}
                type="password"
                onChange={(e) => setSecret(e.target.value)}
                placeholder=""
              />
            </div>
            <Button onClick={scan} disabled={scanning}>
              {scanning
                ? t("probes.clash.scanning", { defaultValue: "扫描中..." })
                : t("probes.clash.scan", { defaultValue: "扫描连接" })}
            </Button>
          </Flex>
          <Text size="1" color="gray">
            {t("probes.clash.hint", {
              defaultValue:
                "运行 Clash 的机器需要开放 external-controller，secret 在 config.yaml 的 secret 字段。常见值：http://127.0.0.1:9090（本机）",
            })}
          </Text>
        </Flex>
      </Card>

      {targets.length > 0 && (
        <>
          <Card>
            <Flex justify="between" align="center" mb="2">
              <Text size="2" weight="medium">
                {t("probes.clash.results", { defaultValue: "扫描结果" })}
                <Text size="1" color="gray" ml="2">
                  ({selected.size}/{targets.length})
                </Text>
              </Text>
              <Flex gap="2">
                <Button
                  size="1"
                  variant="soft"
                  onClick={() =>
                    setSelected(
                      new Set(targets.map((t) => `${t.type}|${t.target}`)),
                    )
                  }
                >
                  {t("common.selectAll", { defaultValue: "全选" })}
                </Button>
                <Button
                  size="1"
                  variant="soft"
                  color="gray"
                  onClick={() => setSelected(new Set())}
                >
                  {t("common.clear", { defaultValue: "清空" })}
                </Button>
              </Flex>
            </Flex>
            <div
              style={{
                maxHeight: 360,
                overflowY: "auto",
              }}
            >
              {targets.map((tgt) => {
                const key = `${tgt.type}|${tgt.target}`;
                return (
                  <label
                    key={key}
                    style={{
                      display: "flex",
                      alignItems: "center",
                      gap: 8,
                      padding: "6px 4px",
                      borderBottom: "1px solid var(--gray-3)",
                      cursor: "pointer",
                    }}
                  >
                    <Checkbox
                      checked={selected.has(key)}
                      onCheckedChange={() => toggleSelect(key)}
                    />
                    <Badge color="indigo" variant="soft" size="1">
                      {tgt.type.toUpperCase()}
                    </Badge>
                    <Text size="2" weight="medium" style={{ flex: 1 }}>
                      {tgt.target}
                    </Text>
                    {tgt.process && (
                      <Badge variant="soft" size="1">
                        {tgt.process}
                      </Badge>
                    )}
                    {tgt.rule && (
                      <Text size="1" color="gray" className="truncate" style={{ maxWidth: 140 }}>
                        {tgt.rule}
                      </Text>
                    )}
                    <Text size="1" color="gray">
                      ×{tgt.count}
                    </Text>
                  </label>
                );
              })}
            </div>
          </Card>

          <Card>
            <Flex direction="column" gap="3">
              <Flex gap="3" wrap="wrap">
                <div style={{ width: 120 }}>
                  <Text as="label" size="2" weight="medium">
                    {t("probes.fields.interval", { defaultValue: "间隔（秒）" })}
                  </Text>
                  <TextField.Root
                    type="number"
                    min={5}
                    value={interval}
                    onChange={(e) =>
                      setInterval(
                        Math.max(5, parseInt(e.target.value, 10) || 60),
                      )
                    }
                  />
                </div>
                <div style={{ width: 220 }}>
                  <Text as="label" size="2" weight="medium">
                    {t("probes.fields.cover", { defaultValue: "覆盖范围" })}
                  </Text>
                  <Select.Root
                    value={cover}
                    onValueChange={(v) => setCover(v as CoverMode)}
                  >
                    <Select.Trigger style={{ width: "100%" }} />
                    <Select.Content>
                      <Select.Item value="all">
                        {t("probes.cover.all", { defaultValue: "全部节点（自动包含新加节点）" })}
                      </Select.Item>
                      <Select.Item value="include">
                        {t("probes.cover.include", { defaultValue: "仅指定节点" })}
                      </Select.Item>
                      <Select.Item value="exclude">
                        {t("probes.cover.exclude", { defaultValue: "排除指定节点" })}
                      </Select.Item>
                    </Select.Content>
                  </Select.Root>
                </div>
              </Flex>
              {cover !== "all" && (
              <div>
                <Flex justify="between" align="center" mb="2">
                  <Text size="2" weight="medium">
                    {cover === "include"
                      ? t("probes.fields.clients", { defaultValue: "目标节点" })
                      : t("probes.fields.excluded_clients", { defaultValue: "排除节点" })}
                    <Text size="1" color="gray" ml="2">
                      ({selectedClients.size}/{nodeList?.length ?? 0})
                    </Text>
                  </Text>
                  <Flex gap="2">
                    <Button
                      size="1"
                      variant="soft"
                      onClick={() =>
                        setSelectedClients(
                          new Set(nodeList?.map((n) => n.uuid) ?? []),
                        )
                      }
                      type="button"
                    >
                      {t("common.selectAll", { defaultValue: "全选" })}
                    </Button>
                    <Button
                      size="1"
                      variant="soft"
                      color="gray"
                      onClick={() => setSelectedClients(new Set())}
                      type="button"
                    >
                      {t("common.clear", { defaultValue: "清空" })}
                    </Button>
                  </Flex>
                </Flex>
                <div
                  style={{
                    maxHeight: 160,
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
                        onCheckedChange={() => toggleClient(n.uuid)}
                      />
                      <span className="text-sm truncate">{n.name}</span>
                    </label>
                  ))}
                </div>
              </div>
              )}
              <Flex justify="end">
                <Button onClick={submit} disabled={submitting}>
                  {submitting
                    ? t("common.submitting", { defaultValue: "提交中..." })
                    : t("probes.clash.add_selected", {
                        defaultValue: `添加选中 ${selected.size} 个`,
                        count: selected.size,
                      })}
                </Button>
              </Flex>
            </Flex>
          </Card>
        </>
      )}
    </Flex>
  );
};

export default ClashHelper;
