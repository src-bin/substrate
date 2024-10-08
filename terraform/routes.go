package terraform

type Route struct {
	Commented                                                                      bool // set by a command-line flag to control costs incurred by NAT Gateways
	DestinationIPv4, DestinationIPv6                                               Value
	EgressOnlyInternetGatewayId, InternetGatewayId, NATGatewayId, TransitGatewayId Value
	Label                                                                          Value
	Provider                                                                       ProviderAlias
	RouteTableId                                                                   Value
}

func (r Route) Ref() Value {
	return Uf("aws_route.%s", r.Label)
}

func (Route) Template() string {
	return `{{if .Commented -}}
/* commented because substrate.nat-gateways contains "no"
{{end -}}
resource "aws_route" {{.Label.Value}} {
{{- if .DestinationIPv4}}
  destination_cidr_block = {{.DestinationIPv4.Value}}
{{- end}}
{{- if .DestinationIPv6}}
  destination_ipv6_cidr_block = {{.DestinationIPv6.Value}}
{{- end}}
{{- if .EgressOnlyInternetGatewayId}}
  egress_only_gateway_id = {{.EgressOnlyInternetGatewayId.Value}}
{{- end}}
{{- if .InternetGatewayId}}
  gateway_id = {{.InternetGatewayId.Value}}
{{- end}}
{{- if .NATGatewayId}}
  nat_gateway_id = {{.NATGatewayId.Value}}
{{- end}}
{{- if .Provider}}
  provider = {{.Provider}}
{{- end}}
  route_table_id = {{.RouteTableId.Value}}
{{- if .TransitGatewayId}}
  transit_gateway_id = {{.TransitGatewayId.Value}}
{{- end}}
}
{{- if .Commented}}
*/
{{- end}}`
}

type RouteTable struct {
	Label    Value
	Provider ProviderAlias
	Tags     Tags
	VpcId    Value
}

func (rt RouteTable) Ref() Value {
	return Uf("aws_route_table.%s", rt.Label)
}

func (RouteTable) Template() string {
	return `resource "aws_route_table" {{.Label.Value}} {
{{- if .Provider}}
  provider = {{.Provider}}
{{- end}}
  tags = {{.Tags.Value}}
  vpc_id = {{.VpcId.Value}}
}`
}

type RouteTableAssociation struct {
	Label                  Value
	Provider               ProviderAlias
	RouteTableId, SubnetId Value
}

func (rta RouteTableAssociation) Ref() Value {
	return Uf("aws_route_table_association.%s", rta.Label)
}

func (RouteTableAssociation) Template() string {
	return `resource "aws_route_table_association" {{.Label.Value}} {
{{- if .Provider}}
  provider = {{.Provider}}
{{- end}}
  route_table_id = {{.RouteTableId.Value}}
  subnet_id = {{.SubnetId.Value}}
}`
}
