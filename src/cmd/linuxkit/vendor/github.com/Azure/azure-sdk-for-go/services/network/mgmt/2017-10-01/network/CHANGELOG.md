Generated from https://github.com/Azure/azure-rest-api-specs/tree/3c764635e7d442b3e74caf593029fcd440b3ef82/specification/network/resource-manager/readme.md tag: `package-2017-10`

Code generator @microsoft.azure/autorest.go@2.1.168

## Breaking Changes

### Removed Funcs

1. *ApplicationGatewaysBackendHealthFuture.Result(ApplicationGatewaysClient) (ApplicationGatewayBackendHealth, error)
1. *ApplicationGatewaysCreateOrUpdateFuture.Result(ApplicationGatewaysClient) (ApplicationGateway, error)
1. *ApplicationGatewaysDeleteFuture.Result(ApplicationGatewaysClient) (autorest.Response, error)
1. *ApplicationGatewaysStartFuture.Result(ApplicationGatewaysClient) (autorest.Response, error)
1. *ApplicationGatewaysStopFuture.Result(ApplicationGatewaysClient) (autorest.Response, error)
1. *ApplicationGatewaysUpdateTagsFuture.Result(ApplicationGatewaysClient) (ApplicationGateway, error)
1. *ApplicationSecurityGroupsCreateOrUpdateFuture.Result(ApplicationSecurityGroupsClient) (ApplicationSecurityGroup, error)
1. *ApplicationSecurityGroupsDeleteFuture.Result(ApplicationSecurityGroupsClient) (autorest.Response, error)
1. *ConnectionMonitorsCreateOrUpdateFuture.Result(ConnectionMonitorsClient) (ConnectionMonitorResult, error)
1. *ConnectionMonitorsDeleteFuture.Result(ConnectionMonitorsClient) (autorest.Response, error)
1. *ConnectionMonitorsQueryFuture.Result(ConnectionMonitorsClient) (ConnectionMonitorQueryResult, error)
1. *ConnectionMonitorsStartFuture.Result(ConnectionMonitorsClient) (autorest.Response, error)
1. *ConnectionMonitorsStopFuture.Result(ConnectionMonitorsClient) (autorest.Response, error)
1. *ExpressRouteCircuitAuthorizationsCreateOrUpdateFuture.Result(ExpressRouteCircuitAuthorizationsClient) (ExpressRouteCircuitAuthorization, error)
1. *ExpressRouteCircuitAuthorizationsDeleteFuture.Result(ExpressRouteCircuitAuthorizationsClient) (autorest.Response, error)
1. *ExpressRouteCircuitPeeringsCreateOrUpdateFuture.Result(ExpressRouteCircuitPeeringsClient) (ExpressRouteCircuitPeering, error)
1. *ExpressRouteCircuitPeeringsDeleteFuture.Result(ExpressRouteCircuitPeeringsClient) (autorest.Response, error)
1. *ExpressRouteCircuitsCreateOrUpdateFuture.Result(ExpressRouteCircuitsClient) (ExpressRouteCircuit, error)
1. *ExpressRouteCircuitsDeleteFuture.Result(ExpressRouteCircuitsClient) (autorest.Response, error)
1. *ExpressRouteCircuitsListArpTableFuture.Result(ExpressRouteCircuitsClient) (ExpressRouteCircuitsArpTableListResult, error)
1. *ExpressRouteCircuitsListRoutesTableFuture.Result(ExpressRouteCircuitsClient) (ExpressRouteCircuitsRoutesTableListResult, error)
1. *ExpressRouteCircuitsListRoutesTableSummaryFuture.Result(ExpressRouteCircuitsClient) (ExpressRouteCircuitsRoutesTableSummaryListResult, error)
1. *ExpressRouteCircuitsUpdateTagsFuture.Result(ExpressRouteCircuitsClient) (ExpressRouteCircuit, error)
1. *InboundNatRulesCreateOrUpdateFuture.Result(InboundNatRulesClient) (InboundNatRule, error)
1. *InboundNatRulesDeleteFuture.Result(InboundNatRulesClient) (autorest.Response, error)
1. *InterfacesCreateOrUpdateFuture.Result(InterfacesClient) (Interface, error)
1. *InterfacesDeleteFuture.Result(InterfacesClient) (autorest.Response, error)
1. *InterfacesGetEffectiveRouteTableFuture.Result(InterfacesClient) (EffectiveRouteListResult, error)
1. *InterfacesListEffectiveNetworkSecurityGroupsFuture.Result(InterfacesClient) (EffectiveNetworkSecurityGroupListResult, error)
1. *InterfacesUpdateTagsFuture.Result(InterfacesClient) (Interface, error)
1. *LoadBalancersCreateOrUpdateFuture.Result(LoadBalancersClient) (LoadBalancer, error)
1. *LoadBalancersDeleteFuture.Result(LoadBalancersClient) (autorest.Response, error)
1. *LoadBalancersUpdateTagsFuture.Result(LoadBalancersClient) (LoadBalancer, error)
1. *LocalNetworkGatewaysCreateOrUpdateFuture.Result(LocalNetworkGatewaysClient) (LocalNetworkGateway, error)
1. *LocalNetworkGatewaysDeleteFuture.Result(LocalNetworkGatewaysClient) (autorest.Response, error)
1. *LocalNetworkGatewaysUpdateTagsFuture.Result(LocalNetworkGatewaysClient) (LocalNetworkGateway, error)
1. *PacketCapturesCreateFuture.Result(PacketCapturesClient) (PacketCaptureResult, error)
1. *PacketCapturesDeleteFuture.Result(PacketCapturesClient) (autorest.Response, error)
1. *PacketCapturesGetStatusFuture.Result(PacketCapturesClient) (PacketCaptureQueryStatusResult, error)
1. *PacketCapturesStopFuture.Result(PacketCapturesClient) (autorest.Response, error)
1. *PublicIPAddressesCreateOrUpdateFuture.Result(PublicIPAddressesClient) (PublicIPAddress, error)
1. *PublicIPAddressesDeleteFuture.Result(PublicIPAddressesClient) (autorest.Response, error)
1. *PublicIPAddressesUpdateTagsFuture.Result(PublicIPAddressesClient) (PublicIPAddress, error)
1. *RouteFilterRulesCreateOrUpdateFuture.Result(RouteFilterRulesClient) (RouteFilterRule, error)
1. *RouteFilterRulesDeleteFuture.Result(RouteFilterRulesClient) (autorest.Response, error)
1. *RouteFilterRulesUpdateFuture.Result(RouteFilterRulesClient) (RouteFilterRule, error)
1. *RouteFiltersCreateOrUpdateFuture.Result(RouteFiltersClient) (RouteFilter, error)
1. *RouteFiltersDeleteFuture.Result(RouteFiltersClient) (autorest.Response, error)
1. *RouteFiltersUpdateFuture.Result(RouteFiltersClient) (RouteFilter, error)
1. *RouteTablesCreateOrUpdateFuture.Result(RouteTablesClient) (RouteTable, error)
1. *RouteTablesDeleteFuture.Result(RouteTablesClient) (autorest.Response, error)
1. *RouteTablesUpdateTagsFuture.Result(RouteTablesClient) (RouteTable, error)
1. *RoutesCreateOrUpdateFuture.Result(RoutesClient) (Route, error)
1. *RoutesDeleteFuture.Result(RoutesClient) (autorest.Response, error)
1. *SecurityGroupsCreateOrUpdateFuture.Result(SecurityGroupsClient) (SecurityGroup, error)
1. *SecurityGroupsDeleteFuture.Result(SecurityGroupsClient) (autorest.Response, error)
1. *SecurityGroupsUpdateTagsFuture.Result(SecurityGroupsClient) (SecurityGroup, error)
1. *SecurityRulesCreateOrUpdateFuture.Result(SecurityRulesClient) (SecurityRule, error)
1. *SecurityRulesDeleteFuture.Result(SecurityRulesClient) (autorest.Response, error)
1. *SubnetsCreateOrUpdateFuture.Result(SubnetsClient) (Subnet, error)
1. *SubnetsDeleteFuture.Result(SubnetsClient) (autorest.Response, error)
1. *VirtualNetworkGatewayConnectionsCreateOrUpdateFuture.Result(VirtualNetworkGatewayConnectionsClient) (VirtualNetworkGatewayConnection, error)
1. *VirtualNetworkGatewayConnectionsDeleteFuture.Result(VirtualNetworkGatewayConnectionsClient) (autorest.Response, error)
1. *VirtualNetworkGatewayConnectionsResetSharedKeyFuture.Result(VirtualNetworkGatewayConnectionsClient) (ConnectionResetSharedKey, error)
1. *VirtualNetworkGatewayConnectionsSetSharedKeyFuture.Result(VirtualNetworkGatewayConnectionsClient) (ConnectionSharedKey, error)
1. *VirtualNetworkGatewayConnectionsUpdateTagsFuture.Result(VirtualNetworkGatewayConnectionsClient) (VirtualNetworkGatewayConnectionListEntity, error)
1. *VirtualNetworkGatewaysCreateOrUpdateFuture.Result(VirtualNetworkGatewaysClient) (VirtualNetworkGateway, error)
1. *VirtualNetworkGatewaysDeleteFuture.Result(VirtualNetworkGatewaysClient) (autorest.Response, error)
1. *VirtualNetworkGatewaysGenerateVpnProfileFuture.Result(VirtualNetworkGatewaysClient) (String, error)
1. *VirtualNetworkGatewaysGeneratevpnclientpackageFuture.Result(VirtualNetworkGatewaysClient) (String, error)
1. *VirtualNetworkGatewaysGetAdvertisedRoutesFuture.Result(VirtualNetworkGatewaysClient) (GatewayRouteListResult, error)
1. *VirtualNetworkGatewaysGetBgpPeerStatusFuture.Result(VirtualNetworkGatewaysClient) (BgpPeerStatusListResult, error)
1. *VirtualNetworkGatewaysGetLearnedRoutesFuture.Result(VirtualNetworkGatewaysClient) (GatewayRouteListResult, error)
1. *VirtualNetworkGatewaysGetVpnProfilePackageURLFuture.Result(VirtualNetworkGatewaysClient) (String, error)
1. *VirtualNetworkGatewaysResetFuture.Result(VirtualNetworkGatewaysClient) (VirtualNetworkGateway, error)
1. *VirtualNetworkGatewaysUpdateTagsFuture.Result(VirtualNetworkGatewaysClient) (VirtualNetworkGateway, error)
1. *VirtualNetworkPeeringsCreateOrUpdateFuture.Result(VirtualNetworkPeeringsClient) (VirtualNetworkPeering, error)
1. *VirtualNetworkPeeringsDeleteFuture.Result(VirtualNetworkPeeringsClient) (autorest.Response, error)
1. *VirtualNetworksCreateOrUpdateFuture.Result(VirtualNetworksClient) (VirtualNetwork, error)
1. *VirtualNetworksDeleteFuture.Result(VirtualNetworksClient) (autorest.Response, error)
1. *VirtualNetworksUpdateTagsFuture.Result(VirtualNetworksClient) (VirtualNetwork, error)
1. *WatchersCheckConnectivityFuture.Result(WatchersClient) (ConnectivityInformation, error)
1. *WatchersDeleteFuture.Result(WatchersClient) (autorest.Response, error)
1. *WatchersGetAzureReachabilityReportFuture.Result(WatchersClient) (AzureReachabilityReport, error)
1. *WatchersGetFlowLogStatusFuture.Result(WatchersClient) (FlowLogInformation, error)
1. *WatchersGetNextHopFuture.Result(WatchersClient) (NextHopResult, error)
1. *WatchersGetTroubleshootingFuture.Result(WatchersClient) (TroubleshootingResult, error)
1. *WatchersGetTroubleshootingResultFuture.Result(WatchersClient) (TroubleshootingResult, error)
1. *WatchersGetVMSecurityRulesFuture.Result(WatchersClient) (SecurityGroupViewResult, error)
1. *WatchersListAvailableProvidersFuture.Result(WatchersClient) (AvailableProvidersList, error)
1. *WatchersSetFlowLogConfigurationFuture.Result(WatchersClient) (FlowLogInformation, error)
1. *WatchersVerifyIPFlowFuture.Result(WatchersClient) (VerificationIPFlowResult, error)

## Struct Changes

### Removed Struct Fields

1. ApplicationGatewaysBackendHealthFuture.azure.Future
1. ApplicationGatewaysCreateOrUpdateFuture.azure.Future
1. ApplicationGatewaysDeleteFuture.azure.Future
1. ApplicationGatewaysStartFuture.azure.Future
1. ApplicationGatewaysStopFuture.azure.Future
1. ApplicationGatewaysUpdateTagsFuture.azure.Future
1. ApplicationSecurityGroupsCreateOrUpdateFuture.azure.Future
1. ApplicationSecurityGroupsDeleteFuture.azure.Future
1. ConnectionMonitorsCreateOrUpdateFuture.azure.Future
1. ConnectionMonitorsDeleteFuture.azure.Future
1. ConnectionMonitorsQueryFuture.azure.Future
1. ConnectionMonitorsStartFuture.azure.Future
1. ConnectionMonitorsStopFuture.azure.Future
1. ExpressRouteCircuitAuthorizationsCreateOrUpdateFuture.azure.Future
1. ExpressRouteCircuitAuthorizationsDeleteFuture.azure.Future
1. ExpressRouteCircuitPeeringsCreateOrUpdateFuture.azure.Future
1. ExpressRouteCircuitPeeringsDeleteFuture.azure.Future
1. ExpressRouteCircuitsCreateOrUpdateFuture.azure.Future
1. ExpressRouteCircuitsDeleteFuture.azure.Future
1. ExpressRouteCircuitsListArpTableFuture.azure.Future
1. ExpressRouteCircuitsListRoutesTableFuture.azure.Future
1. ExpressRouteCircuitsListRoutesTableSummaryFuture.azure.Future
1. ExpressRouteCircuitsUpdateTagsFuture.azure.Future
1. InboundNatRulesCreateOrUpdateFuture.azure.Future
1. InboundNatRulesDeleteFuture.azure.Future
1. InterfacesCreateOrUpdateFuture.azure.Future
1. InterfacesDeleteFuture.azure.Future
1. InterfacesGetEffectiveRouteTableFuture.azure.Future
1. InterfacesListEffectiveNetworkSecurityGroupsFuture.azure.Future
1. InterfacesUpdateTagsFuture.azure.Future
1. LoadBalancersCreateOrUpdateFuture.azure.Future
1. LoadBalancersDeleteFuture.azure.Future
1. LoadBalancersUpdateTagsFuture.azure.Future
1. LocalNetworkGatewaysCreateOrUpdateFuture.azure.Future
1. LocalNetworkGatewaysDeleteFuture.azure.Future
1. LocalNetworkGatewaysUpdateTagsFuture.azure.Future
1. PacketCapturesCreateFuture.azure.Future
1. PacketCapturesDeleteFuture.azure.Future
1. PacketCapturesGetStatusFuture.azure.Future
1. PacketCapturesStopFuture.azure.Future
1. PublicIPAddressesCreateOrUpdateFuture.azure.Future
1. PublicIPAddressesDeleteFuture.azure.Future
1. PublicIPAddressesUpdateTagsFuture.azure.Future
1. RouteFilterRulesCreateOrUpdateFuture.azure.Future
1. RouteFilterRulesDeleteFuture.azure.Future
1. RouteFilterRulesUpdateFuture.azure.Future
1. RouteFiltersCreateOrUpdateFuture.azure.Future
1. RouteFiltersDeleteFuture.azure.Future
1. RouteFiltersUpdateFuture.azure.Future
1. RouteTablesCreateOrUpdateFuture.azure.Future
1. RouteTablesDeleteFuture.azure.Future
1. RouteTablesUpdateTagsFuture.azure.Future
1. RoutesCreateOrUpdateFuture.azure.Future
1. RoutesDeleteFuture.azure.Future
1. SecurityGroupsCreateOrUpdateFuture.azure.Future
1. SecurityGroupsDeleteFuture.azure.Future
1. SecurityGroupsUpdateTagsFuture.azure.Future
1. SecurityRulesCreateOrUpdateFuture.azure.Future
1. SecurityRulesDeleteFuture.azure.Future
1. SubnetsCreateOrUpdateFuture.azure.Future
1. SubnetsDeleteFuture.azure.Future
1. VirtualNetworkGatewayConnectionsCreateOrUpdateFuture.azure.Future
1. VirtualNetworkGatewayConnectionsDeleteFuture.azure.Future
1. VirtualNetworkGatewayConnectionsResetSharedKeyFuture.azure.Future
1. VirtualNetworkGatewayConnectionsSetSharedKeyFuture.azure.Future
1. VirtualNetworkGatewayConnectionsUpdateTagsFuture.azure.Future
1. VirtualNetworkGatewaysCreateOrUpdateFuture.azure.Future
1. VirtualNetworkGatewaysDeleteFuture.azure.Future
1. VirtualNetworkGatewaysGenerateVpnProfileFuture.azure.Future
1. VirtualNetworkGatewaysGeneratevpnclientpackageFuture.azure.Future
1. VirtualNetworkGatewaysGetAdvertisedRoutesFuture.azure.Future
1. VirtualNetworkGatewaysGetBgpPeerStatusFuture.azure.Future
1. VirtualNetworkGatewaysGetLearnedRoutesFuture.azure.Future
1. VirtualNetworkGatewaysGetVpnProfilePackageURLFuture.azure.Future
1. VirtualNetworkGatewaysResetFuture.azure.Future
1. VirtualNetworkGatewaysUpdateTagsFuture.azure.Future
1. VirtualNetworkPeeringsCreateOrUpdateFuture.azure.Future
1. VirtualNetworkPeeringsDeleteFuture.azure.Future
1. VirtualNetworksCreateOrUpdateFuture.azure.Future
1. VirtualNetworksDeleteFuture.azure.Future
1. VirtualNetworksUpdateTagsFuture.azure.Future
1. WatchersCheckConnectivityFuture.azure.Future
1. WatchersDeleteFuture.azure.Future
1. WatchersGetAzureReachabilityReportFuture.azure.Future
1. WatchersGetFlowLogStatusFuture.azure.Future
1. WatchersGetNextHopFuture.azure.Future
1. WatchersGetTroubleshootingFuture.azure.Future
1. WatchersGetTroubleshootingResultFuture.azure.Future
1. WatchersGetVMSecurityRulesFuture.azure.Future
1. WatchersListAvailableProvidersFuture.azure.Future
1. WatchersSetFlowLogConfigurationFuture.azure.Future
1. WatchersVerifyIPFlowFuture.azure.Future

## Struct Changes

### New Struct Fields

1. ApplicationGatewaysBackendHealthFuture.Result
1. ApplicationGatewaysBackendHealthFuture.azure.FutureAPI
1. ApplicationGatewaysCreateOrUpdateFuture.Result
1. ApplicationGatewaysCreateOrUpdateFuture.azure.FutureAPI
1. ApplicationGatewaysDeleteFuture.Result
1. ApplicationGatewaysDeleteFuture.azure.FutureAPI
1. ApplicationGatewaysStartFuture.Result
1. ApplicationGatewaysStartFuture.azure.FutureAPI
1. ApplicationGatewaysStopFuture.Result
1. ApplicationGatewaysStopFuture.azure.FutureAPI
1. ApplicationGatewaysUpdateTagsFuture.Result
1. ApplicationGatewaysUpdateTagsFuture.azure.FutureAPI
1. ApplicationSecurityGroupsCreateOrUpdateFuture.Result
1. ApplicationSecurityGroupsCreateOrUpdateFuture.azure.FutureAPI
1. ApplicationSecurityGroupsDeleteFuture.Result
1. ApplicationSecurityGroupsDeleteFuture.azure.FutureAPI
1. ConnectionMonitorsCreateOrUpdateFuture.Result
1. ConnectionMonitorsCreateOrUpdateFuture.azure.FutureAPI
1. ConnectionMonitorsDeleteFuture.Result
1. ConnectionMonitorsDeleteFuture.azure.FutureAPI
1. ConnectionMonitorsQueryFuture.Result
1. ConnectionMonitorsQueryFuture.azure.FutureAPI
1. ConnectionMonitorsStartFuture.Result
1. ConnectionMonitorsStartFuture.azure.FutureAPI
1. ConnectionMonitorsStopFuture.Result
1. ConnectionMonitorsStopFuture.azure.FutureAPI
1. ExpressRouteCircuitAuthorizationsCreateOrUpdateFuture.Result
1. ExpressRouteCircuitAuthorizationsCreateOrUpdateFuture.azure.FutureAPI
1. ExpressRouteCircuitAuthorizationsDeleteFuture.Result
1. ExpressRouteCircuitAuthorizationsDeleteFuture.azure.FutureAPI
1. ExpressRouteCircuitPeeringsCreateOrUpdateFuture.Result
1. ExpressRouteCircuitPeeringsCreateOrUpdateFuture.azure.FutureAPI
1. ExpressRouteCircuitPeeringsDeleteFuture.Result
1. ExpressRouteCircuitPeeringsDeleteFuture.azure.FutureAPI
1. ExpressRouteCircuitsCreateOrUpdateFuture.Result
1. ExpressRouteCircuitsCreateOrUpdateFuture.azure.FutureAPI
1. ExpressRouteCircuitsDeleteFuture.Result
1. ExpressRouteCircuitsDeleteFuture.azure.FutureAPI
1. ExpressRouteCircuitsListArpTableFuture.Result
1. ExpressRouteCircuitsListArpTableFuture.azure.FutureAPI
1. ExpressRouteCircuitsListRoutesTableFuture.Result
1. ExpressRouteCircuitsListRoutesTableFuture.azure.FutureAPI
1. ExpressRouteCircuitsListRoutesTableSummaryFuture.Result
1. ExpressRouteCircuitsListRoutesTableSummaryFuture.azure.FutureAPI
1. ExpressRouteCircuitsUpdateTagsFuture.Result
1. ExpressRouteCircuitsUpdateTagsFuture.azure.FutureAPI
1. InboundNatRulesCreateOrUpdateFuture.Result
1. InboundNatRulesCreateOrUpdateFuture.azure.FutureAPI
1. InboundNatRulesDeleteFuture.Result
1. InboundNatRulesDeleteFuture.azure.FutureAPI
1. InterfacesCreateOrUpdateFuture.Result
1. InterfacesCreateOrUpdateFuture.azure.FutureAPI
1. InterfacesDeleteFuture.Result
1. InterfacesDeleteFuture.azure.FutureAPI
1. InterfacesGetEffectiveRouteTableFuture.Result
1. InterfacesGetEffectiveRouteTableFuture.azure.FutureAPI
1. InterfacesListEffectiveNetworkSecurityGroupsFuture.Result
1. InterfacesListEffectiveNetworkSecurityGroupsFuture.azure.FutureAPI
1. InterfacesUpdateTagsFuture.Result
1. InterfacesUpdateTagsFuture.azure.FutureAPI
1. LoadBalancersCreateOrUpdateFuture.Result
1. LoadBalancersCreateOrUpdateFuture.azure.FutureAPI
1. LoadBalancersDeleteFuture.Result
1. LoadBalancersDeleteFuture.azure.FutureAPI
1. LoadBalancersUpdateTagsFuture.Result
1. LoadBalancersUpdateTagsFuture.azure.FutureAPI
1. LocalNetworkGatewaysCreateOrUpdateFuture.Result
1. LocalNetworkGatewaysCreateOrUpdateFuture.azure.FutureAPI
1. LocalNetworkGatewaysDeleteFuture.Result
1. LocalNetworkGatewaysDeleteFuture.azure.FutureAPI
1. LocalNetworkGatewaysUpdateTagsFuture.Result
1. LocalNetworkGatewaysUpdateTagsFuture.azure.FutureAPI
1. PacketCapturesCreateFuture.Result
1. PacketCapturesCreateFuture.azure.FutureAPI
1. PacketCapturesDeleteFuture.Result
1. PacketCapturesDeleteFuture.azure.FutureAPI
1. PacketCapturesGetStatusFuture.Result
1. PacketCapturesGetStatusFuture.azure.FutureAPI
1. PacketCapturesStopFuture.Result
1. PacketCapturesStopFuture.azure.FutureAPI
1. PublicIPAddressesCreateOrUpdateFuture.Result
1. PublicIPAddressesCreateOrUpdateFuture.azure.FutureAPI
1. PublicIPAddressesDeleteFuture.Result
1. PublicIPAddressesDeleteFuture.azure.FutureAPI
1. PublicIPAddressesUpdateTagsFuture.Result
1. PublicIPAddressesUpdateTagsFuture.azure.FutureAPI
1. RouteFilterRulesCreateOrUpdateFuture.Result
1. RouteFilterRulesCreateOrUpdateFuture.azure.FutureAPI
1. RouteFilterRulesDeleteFuture.Result
1. RouteFilterRulesDeleteFuture.azure.FutureAPI
1. RouteFilterRulesUpdateFuture.Result
1. RouteFilterRulesUpdateFuture.azure.FutureAPI
1. RouteFiltersCreateOrUpdateFuture.Result
1. RouteFiltersCreateOrUpdateFuture.azure.FutureAPI
1. RouteFiltersDeleteFuture.Result
1. RouteFiltersDeleteFuture.azure.FutureAPI
1. RouteFiltersUpdateFuture.Result
1. RouteFiltersUpdateFuture.azure.FutureAPI
1. RouteTablesCreateOrUpdateFuture.Result
1. RouteTablesCreateOrUpdateFuture.azure.FutureAPI
1. RouteTablesDeleteFuture.Result
1. RouteTablesDeleteFuture.azure.FutureAPI
1. RouteTablesUpdateTagsFuture.Result
1. RouteTablesUpdateTagsFuture.azure.FutureAPI
1. RoutesCreateOrUpdateFuture.Result
1. RoutesCreateOrUpdateFuture.azure.FutureAPI
1. RoutesDeleteFuture.Result
1. RoutesDeleteFuture.azure.FutureAPI
1. SecurityGroupsCreateOrUpdateFuture.Result
1. SecurityGroupsCreateOrUpdateFuture.azure.FutureAPI
1. SecurityGroupsDeleteFuture.Result
1. SecurityGroupsDeleteFuture.azure.FutureAPI
1. SecurityGroupsUpdateTagsFuture.Result
1. SecurityGroupsUpdateTagsFuture.azure.FutureAPI
1. SecurityRulesCreateOrUpdateFuture.Result
1. SecurityRulesCreateOrUpdateFuture.azure.FutureAPI
1. SecurityRulesDeleteFuture.Result
1. SecurityRulesDeleteFuture.azure.FutureAPI
1. SubnetsCreateOrUpdateFuture.Result
1. SubnetsCreateOrUpdateFuture.azure.FutureAPI
1. SubnetsDeleteFuture.Result
1. SubnetsDeleteFuture.azure.FutureAPI
1. VirtualNetworkGatewayConnectionsCreateOrUpdateFuture.Result
1. VirtualNetworkGatewayConnectionsCreateOrUpdateFuture.azure.FutureAPI
1. VirtualNetworkGatewayConnectionsDeleteFuture.Result
1. VirtualNetworkGatewayConnectionsDeleteFuture.azure.FutureAPI
1. VirtualNetworkGatewayConnectionsResetSharedKeyFuture.Result
1. VirtualNetworkGatewayConnectionsResetSharedKeyFuture.azure.FutureAPI
1. VirtualNetworkGatewayConnectionsSetSharedKeyFuture.Result
1. VirtualNetworkGatewayConnectionsSetSharedKeyFuture.azure.FutureAPI
1. VirtualNetworkGatewayConnectionsUpdateTagsFuture.Result
1. VirtualNetworkGatewayConnectionsUpdateTagsFuture.azure.FutureAPI
1. VirtualNetworkGatewaysCreateOrUpdateFuture.Result
1. VirtualNetworkGatewaysCreateOrUpdateFuture.azure.FutureAPI
1. VirtualNetworkGatewaysDeleteFuture.Result
1. VirtualNetworkGatewaysDeleteFuture.azure.FutureAPI
1. VirtualNetworkGatewaysGenerateVpnProfileFuture.Result
1. VirtualNetworkGatewaysGenerateVpnProfileFuture.azure.FutureAPI
1. VirtualNetworkGatewaysGeneratevpnclientpackageFuture.Result
1. VirtualNetworkGatewaysGeneratevpnclientpackageFuture.azure.FutureAPI
1. VirtualNetworkGatewaysGetAdvertisedRoutesFuture.Result
1. VirtualNetworkGatewaysGetAdvertisedRoutesFuture.azure.FutureAPI
1. VirtualNetworkGatewaysGetBgpPeerStatusFuture.Result
1. VirtualNetworkGatewaysGetBgpPeerStatusFuture.azure.FutureAPI
1. VirtualNetworkGatewaysGetLearnedRoutesFuture.Result
1. VirtualNetworkGatewaysGetLearnedRoutesFuture.azure.FutureAPI
1. VirtualNetworkGatewaysGetVpnProfilePackageURLFuture.Result
1. VirtualNetworkGatewaysGetVpnProfilePackageURLFuture.azure.FutureAPI
1. VirtualNetworkGatewaysResetFuture.Result
1. VirtualNetworkGatewaysResetFuture.azure.FutureAPI
1. VirtualNetworkGatewaysUpdateTagsFuture.Result
1. VirtualNetworkGatewaysUpdateTagsFuture.azure.FutureAPI
1. VirtualNetworkPeeringsCreateOrUpdateFuture.Result
1. VirtualNetworkPeeringsCreateOrUpdateFuture.azure.FutureAPI
1. VirtualNetworkPeeringsDeleteFuture.Result
1. VirtualNetworkPeeringsDeleteFuture.azure.FutureAPI
1. VirtualNetworksCreateOrUpdateFuture.Result
1. VirtualNetworksCreateOrUpdateFuture.azure.FutureAPI
1. VirtualNetworksDeleteFuture.Result
1. VirtualNetworksDeleteFuture.azure.FutureAPI
1. VirtualNetworksUpdateTagsFuture.Result
1. VirtualNetworksUpdateTagsFuture.azure.FutureAPI
1. WatchersCheckConnectivityFuture.Result
1. WatchersCheckConnectivityFuture.azure.FutureAPI
1. WatchersDeleteFuture.Result
1. WatchersDeleteFuture.azure.FutureAPI
1. WatchersGetAzureReachabilityReportFuture.Result
1. WatchersGetAzureReachabilityReportFuture.azure.FutureAPI
1. WatchersGetFlowLogStatusFuture.Result
1. WatchersGetFlowLogStatusFuture.azure.FutureAPI
1. WatchersGetNextHopFuture.Result
1. WatchersGetNextHopFuture.azure.FutureAPI
1. WatchersGetTroubleshootingFuture.Result
1. WatchersGetTroubleshootingFuture.azure.FutureAPI
1. WatchersGetTroubleshootingResultFuture.Result
1. WatchersGetTroubleshootingResultFuture.azure.FutureAPI
1. WatchersGetVMSecurityRulesFuture.Result
1. WatchersGetVMSecurityRulesFuture.azure.FutureAPI
1. WatchersListAvailableProvidersFuture.Result
1. WatchersListAvailableProvidersFuture.azure.FutureAPI
1. WatchersSetFlowLogConfigurationFuture.Result
1. WatchersSetFlowLogConfigurationFuture.azure.FutureAPI
1. WatchersVerifyIPFlowFuture.Result
1. WatchersVerifyIPFlowFuture.azure.FutureAPI
