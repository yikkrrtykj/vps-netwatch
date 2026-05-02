import * as React from "react";
import { Card, Tabs, Flex, Text } from "@radix-ui/themes";
import { useTranslation } from "react-i18next";
import { NodeListProvider } from "@/contexts/NodeListContext";
import ProbeManual from "@/components/admin/ProbeManual";
import ClashHelper from "@/components/admin/ClashHelper";

export default function ProbesPage() {
  const { t } = useTranslation();
  const [tab, setTab] = React.useState<"manual" | "clash">("manual");

  return (
    <NodeListProvider>
      <Flex direction="column" gap="4" className="p-4 max-w-5xl mx-auto w-full">
        <div>
          <Text size="5" weight="bold">
            {t("probes.title", { defaultValue: "探针" })}
          </Text>
          <Text as="p" size="2" color="gray" mt="1">
            {t("probes.description", {
              defaultValue:
                "为节点添加 ICMP / TCP / HTTP 探测目标，支持手动添加或从 Clash/mihomo 当前活跃连接中批量导入。",
            })}
          </Text>
        </div>

        <Card>
          <Tabs.Root
            value={tab}
            onValueChange={(v) => setTab(v as "manual" | "clash")}
          >
            <Tabs.List>
              <Tabs.Trigger value="manual">
                {t("probes.tabs.manual", { defaultValue: "手动添加" })}
              </Tabs.Trigger>
              <Tabs.Trigger value="clash">
                {t("probes.tabs.clash", { defaultValue: "Clash 助手" })}
              </Tabs.Trigger>
            </Tabs.List>

            <div className="pt-4">
              <Tabs.Content value="manual">
                <ProbeManual />
              </Tabs.Content>
              <Tabs.Content value="clash">
                <ClashHelper />
              </Tabs.Content>
            </div>
          </Tabs.Root>
        </Card>
      </Flex>
    </NodeListProvider>
  );
}
