package nftables

import (
	"fmt"

	"github.com/evilsocket/opensnitch/daemon/firewall/nftables/exprs"
	"github.com/evilsocket/opensnitch/daemon/log"
	"github.com/google/nftables"
)

// AddTable adds a new table to nftables.
func (n *Nft) AddTable(name, family string) (*nftables.Table, error) {
	famCode := getFamilyCode(family)
	tbl := &nftables.Table{
		Family: famCode,
		Name:   name,
	}
	n.conn.AddTable(tbl)

	if !n.Commit() {
		return nil, fmt.Errorf("%s error adding system firewall table: %s, family: %s (%d)", logTag, name, family, famCode)
	}
	key := getTableKey(name, family)
	sysTables.Add(key, tbl)
	return tbl, nil
}

func (n *Nft) getTable(name, family string) *nftables.Table {
	return sysTables.Get(getTableKey(name, family))
}

func getTableKey(name string, family interface{}) string {
	return fmt.Sprint(name, "-", family)
}

func (n *Nft) addInterceptionTables() error {
	if _, err := n.AddTable(exprs.NFT_CHAIN_MANGLE, exprs.NFT_FAMILY_INET); err != nil {
		return err
	}
	if _, err := n.AddTable(exprs.NFT_CHAIN_FILTER, exprs.NFT_FAMILY_INET); err != nil {
		return err
	}
	return nil
}

// Contrary to iptables, in nftables there're no predefined rules.
// Convention is though to use the iptables names by default.
// We need at least: mangle and filter tables, inet family (IPv4 and IPv6).
func (n *Nft) addSystemTables() {
	n.AddTable(exprs.NFT_CHAIN_MANGLE, exprs.NFT_FAMILY_INET)
	n.AddTable(exprs.NFT_CHAIN_FILTER, exprs.NFT_FAMILY_INET)
}

// return the number of rules that we didn't add.
func (n *Nft) nonSystemRules(tbl *nftables.Table) int {
	chains, err := n.conn.ListChains()
	if err != nil {
		return -1
	}
	t := 0
	for _, c := range chains {
		if tbl.Name != c.Table.Name && tbl.Family != c.Table.Family {
			continue
		}
		rules, err := n.conn.GetRule(c.Table, c)
		if err != nil {
			return -1
		}
		t += len(rules)
	}

	return t
}

func (n *Nft) delSystemTables() {
	for k, tbl := range sysTables.List() {
		if n.nonSystemRules(tbl) != 0 {
			continue
		}
		n.conn.DelTable(tbl)
		if !n.Commit() {
			log.Warning("error deleting system table: %s", k)
			continue
		}
		sysTables.Del(k)
	}
}
