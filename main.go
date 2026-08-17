package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"sort"
	"strings"
	"syscall"
	"time"

	"github.com/bradfitz/gomemcache/memcache"
	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
)

type PingStats struct {
	Host        string          `json:"host"`
	Sent        int             `json:"sent"`
	Received    int             `json:"received"`
	Lost        int             `json:"lost"`
	LossPercent float64         `json:"loss_percent"`
	MinRTT      time.Duration   `json:"min_rtt_ms"`
	MaxRTT      time.Duration   `json:"max_rtt_ms"`
	AvgRTT      time.Duration   `json:"avg_rtt_ms"`
	StdDev      time.Duration   `json:"stddev_ms"`
	Jitter      time.Duration   `json:"jitter_ms"`
	RTTs        []time.Duration `json:"rtts_ms"`
	LastRTT     time.Duration   `json:"last_rtt_ms"`
	ResolveTime time.Duration   `json:"resolve_time_ms"`
	Errors      []string        `json:"errors"`
	Interface   string          `json:"interface"`
	SourceIP    string          `json:"source_ip"`
	Timestamp   time.Time       `json:"timestamp"`
	MOS         float64         `json:"mos"`
}

type MemcachedStore struct {
	client  *memcache.Client
	prefix  string
	ttl     int32
	servers []string
}

func NewMemcachedStore(servers []string, prefix string, ttlSeconds int32) *MemcachedStore {
	if len(servers) == 0 {
		servers = []string{"localhost:11211"}
	}
	return &MemcachedStore{
		client:  memcache.New(servers...),
		prefix:  prefix,
		ttl:     ttlSeconds,
		servers: servers,
	}
}

func (m *MemcachedStore) SavePingStats(stats *PingStats, quietMode bool) error {
	key := stats.Interface
	if key == "" {
		key = stats.SourceIP
	}
	if key == "" {
		key = "default"
	}

	stats.Timestamp = time.Now()

	jsonStats := struct {
		Host        string    `json:"host"`
		Sent        int       `json:"sent"`
		Received    int       `json:"received"`
		Lost        int       `json:"lost"`
		LossPercent float64   `json:"loss_percent"`
		MinRTT      float64   `json:"min_rtt_ms"`
		MaxRTT      float64   `json:"max_rtt_ms"`
		AvgRTT      float64   `json:"avg_rtt_ms"`
		StdDev      float64   `json:"stddev_ms"`
		Jitter      float64   `json:"jitter_ms"`
		RTTs        []float64 `json:"rtts_ms"`
		LastRTT     float64   `json:"last_rtt_ms"`
		ResolveTime float64   `json:"resolve_time_ms"`
		Errors      []string  `json:"errors"`
		Interface   string    `json:"interface"`
		SourceIP    string    `json:"source_ip"`
		Timestamp   time.Time `json:"timestamp"`
		MOS         float64   `json:"mos"`
	}{
		Host:        stats.Host,
		Sent:        stats.Sent,
		Received:    stats.Received,
		Lost:        stats.Lost,
		LossPercent: stats.LossPercent,
		MinRTT:      float64(stats.MinRTT) / float64(time.Millisecond),
		MaxRTT:      float64(stats.MaxRTT) / float64(time.Millisecond),
		AvgRTT:      float64(stats.AvgRTT) / float64(time.Millisecond),
		StdDev:      float64(stats.StdDev) / float64(time.Millisecond),
		Jitter:      float64(stats.Jitter) / float64(time.Millisecond),
		LastRTT:     float64(stats.LastRTT) / float64(time.Millisecond),
		ResolveTime: float64(stats.ResolveTime) / float64(time.Millisecond),
		Errors:      stats.Errors,
		Interface:   stats.Interface,
		SourceIP:    stats.SourceIP,
		Timestamp:   stats.Timestamp,
		MOS:         stats.MOS,
	}

	jsonStats.RTTs = make([]float64, len(stats.RTTs))
	for i, rtt := range stats.RTTs {
		jsonStats.RTTs[i] = float64(rtt) / float64(time.Millisecond)
	}

	data, err := json.Marshal(jsonStats)
	if err != nil {
		return fmt.Errorf("erreur de serialisation JSON : %v", err)
	}

	fullKey := m.prefix + key
	err = m.client.Set(&memcache.Item{
		Key:        fullKey,
		Value:      data,
		Expiration: m.ttl,
	})
	if err != nil {
		return fmt.Errorf("erreur de stockage Memcached : %v", err)
	}

	if !quietMode {
		fmt.Printf("\nStatistiques sauvegardees dans Memcached\n")
		fmt.Printf("   Cle : %s%s\n", m.prefix, key)
		fmt.Printf("   Interface : %s\n", key)
		fmt.Printf("   Timestamp : %s\n", stats.Timestamp.Format("2006-01-02 15:04:05"))
		fmt.Printf("   L'ancienne valeur a ete ecrasee\n")
	}

	return nil
}

// DeletePingStats uses raw TCP for reliable deletion (fixes gomemcache issues)
func (m *MemcachedStore) DeletePingStats(key string, quietMode bool) error {
	fullKey := m.prefix + key

	// Use raw TCP instead of gomemcache client for reliable deletes
	server := m.servers[0]
	conn, err := net.Dial("tcp", server)
	if err != nil {
		return fmt.Errorf("failed to connect to memcached: %v", err)
	}
	defer conn.Close()

	// Send delete command
	cmd := fmt.Sprintf("delete %s\r\n", fullKey)
	_, err = conn.Write([]byte(cmd))
	if err != nil {
		return fmt.Errorf("failed to send delete command: %v", err)
	}

	// Read response
	reader := bufio.NewReader(conn)
	response, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("failed to read response: %v", err)
	}

	response = strings.TrimSpace(response)

	// DELETED = success, NOT_FOUND = already gone (both OK)
	if response != "DELETED" && response != "NOT_FOUND" {
		return fmt.Errorf("unexpected response from memcached: %s", response)
	}

	if !quietMode {
		fmt.Printf("Anciennes donnees pour l'interface '%s' (prefixe '%s') supprimees de Memcached\n", key, m.prefix)
	}

	return nil
}

func createICMPConn(srcIP string, iface string) (net.PacketConn, error) {
	var conn net.PacketConn
	var err error

	if srcIP != "" {
		conn, err = net.ListenPacket("ip4:icmp", srcIP)
	} else {
		conn, err = net.ListenPacket("ip4:icmp", "0.0.0.0")
	}
	if err != nil {
		return nil, fmt.Errorf("ListenPacket failed: %v", err)
	}

	if iface != "" {
		if ipConn, ok := conn.(*net.IPConn); ok {
			file, err := ipConn.File()
			if err != nil {
				conn.Close()
				return nil, fmt.Errorf("failed to get file descriptor: %v", err)
			}

			fd := int(file.Fd())
			err = syscall.SetsockoptString(fd, syscall.SOL_SOCKET, syscall.SO_BINDTODEVICE, iface)
			file.Close()

			if err != nil {
				conn.Close()
				return nil, fmt.Errorf("SO_BINDTODEVICE failed for interface %s: %v", iface, err)
			}
		}
	}

	return conn, nil
}

func pingWithInterface(host string, iface string, sourceIP string, count int, interval time.Duration, quietMode bool, stats *PingStats) error {
	dst, err := net.ResolveIPAddr("ip4", host)
	if err != nil {
		return err
	}

	var src string
	if sourceIP != "" {
		src = sourceIP
	} else if iface != "" {
		srcIP, err := getInterfaceIP(iface)
		if err != nil {
			return err
		}
		src = srcIP
	}

	conn, err := createICMPConn(src, iface)
	if err != nil {
		return fmt.Errorf("failed to create ICMP connection: %v", err)
	}
	defer conn.Close()

	for i := 0; i < count; i++ {
		stats.Sent++

		msg := &icmp.Message{
			Type: ipv4.ICMPTypeEcho,
			Code: 0,
			Body: &icmp.Echo{
				ID:   os.Getpid() & 0xffff,
				Seq:  i,
				Data: []byte("wanqa"),
			},
		}
		data, err := msg.Marshal(nil)
		if err != nil {
			stats.Errors = append(stats.Errors, err.Error())
			continue
		}

		start := time.Now()
		_, err = conn.WriteTo(data, dst)
		if err != nil {
			stats.Errors = append(stats.Errors, err.Error())
			if !quietMode {
				fmt.Printf("[%d/%d] Erreur d'envoi: %v\n", i+1, count, err)
			}
			continue
		}

		reply := make([]byte, 1500)
		conn.SetReadDeadline(time.Now().Add(interval - 10*time.Millisecond))
		n, peer, err := conn.ReadFrom(reply)

		if err != nil {
			stats.Errors = append(stats.Errors, "Delai d'attente depasse")
			if !quietMode {
				fmt.Printf("[%d/%d] Delai d'attente depasse pour seq=%d\n", i+1, count, i)
			}
			continue
		}

		rtt := time.Since(start)
		rm, err := icmp.ParseMessage(ipv4.ICMPTypeEchoReply.Protocol(), reply[:n])
		if err != nil {
			stats.Errors = append(stats.Errors, err.Error())
			continue
		}

		if rm.Type == ipv4.ICMPTypeEchoReply {
			stats.Received++
			stats.LastRTT = rtt
			stats.RTTs = append(stats.RTTs, rtt)
			if !quietMode {
				fmt.Printf("[%d/%d] Reponse de %s : seq=%d temps=%.3f ms\n",
					stats.Received, count, peer.String(), i, float64(rtt)/float64(time.Millisecond))
			}
		} else {
			stats.Errors = append(stats.Errors, fmt.Sprintf("Type ICMP inattendu: %v", rm.Type))
		}

		if i < count-1 {
			time.Sleep(interval - rtt)
		}
	}

	return nil
}

func main() {
	quietMode := false
	args := []string{}

	for _, arg := range os.Args[1:] {
		if arg == "-q" || arg == "--quiet" {
			quietMode = true
		} else {
			args = append(args, arg)
		}
	}

	if len(args) < 1 {
		if !quietMode {
			fmt.Println("Utilisation : sudo ./main [-q|--quiet] <adresse_ip> [nombre] [intervalle_ms] [interface] [memcached_server] [memcached_prefix]")
			fmt.Println("Exemple : sudo ./main 8.8.8.8 10 500 eth0 localhost:11211 wanqa:")
			fmt.Println("          sudo ./main -q 8.8.8.8 10 500 eth0 localhost:11211 lanqa:")
			fmt.Println("\nInterfaces disponibles :")
			listInterfaces()
		}
		os.Exit(1)
	}

	host := args[0]
	count := 4
	interval := 1000 * time.Millisecond
	var iface string
	var sourceIP string
	var memcachedServer string
	memcachedPrefix := "wanqa:" // Default prefix

	if len(args) > 1 {
		if c, err := fmt.Sscanf(args[1], "%d", &count); c == 1 && err == nil {
			if count < 1 {
				count = 4
			}
		}
	}
	if len(args) > 2 {
		var iv int
		if _, err := fmt.Sscanf(args[2], "%d", &iv); err == nil {
			interval = time.Duration(iv) * time.Millisecond
		}
	}
	if len(args) > 3 {
		iface = args[3]
		if iface == "" {
			iface = ""
		} else if net.ParseIP(iface) != nil {
			sourceIP = iface
			iface = ""
		}
	}
	if len(args) > 4 {
		memcachedServer = args[4]
	}
	if len(args) > 5 {
		memcachedPrefix = args[5]
	}

	var mcStore *MemcachedStore
	if memcachedServer != "" {
		mcStore = NewMemcachedStore([]string{memcachedServer}, memcachedPrefix, 3600)
		if !quietMode {
			fmt.Printf("Connexion Memcached etablie : %s\n", memcachedServer)
			fmt.Printf("Les resultats seront stockes avec le prefixe '%s' et la cle = nom de l'interface\n", memcachedPrefix)
			fmt.Println("    (ecrasement de la derniere valeur pour chaque interface)\n")
		}
	}

	stats := &PingStats{
		Host:   host,
		RTTs:   make([]time.Duration, 0, count),
		Errors: make([]string, 0),
	}

	if iface != "" && sourceIP == "" {
		ip, err := getInterfaceIP(iface)
		if err != nil {
			// Interface error (not found or no IPv4) - delete from memcached if exists
			if mcStore != nil {
				delErr := mcStore.DeletePingStats(iface, quietMode)
				if delErr != nil {
					if !quietMode {
						fmt.Printf("Erreur lors de la suppression Memcached : %v\n", delErr)
					}
				}
			}
			if !quietMode {
				fmt.Printf("Erreur lors de la recuperation de l'interface %s : %v\n", iface, err)
				fmt.Println("\nInterfaces disponibles :")
				listInterfaces()
			}
			os.Exit(1)
		}
		sourceIP = ip
		stats.Interface = iface
	} else if sourceIP != "" {
		stats.SourceIP = sourceIP
		foundIface := getInterfaceByIP(sourceIP)
		if foundIface != "" {
			stats.Interface = foundIface
			iface = foundIface
		}
	}

	if !quietMode {
		fmt.Printf("PING %s ", host)
		if stats.Interface != "" {
			fmt.Printf("via %s ", stats.Interface)
		}
		if stats.SourceIP != "" {
			fmt.Printf("(src : %s) ", stats.SourceIP)
		}
		fmt.Printf("(%d paquets, intervalle %v)\n\n", count, interval)

		if stats.Interface != "" {
			fmt.Printf("Liaison stricte a l'interface: %s\n", stats.Interface)
		}
	}

	resolveStart := time.Now()
	_, err := net.ResolveIPAddr("ip4:icmp", host)
	if err != nil {
		if !quietMode {
			fmt.Printf("Echec de la resolution DNS : %v\n", err)
		}
		os.Exit(1)
	}
	stats.ResolveTime = time.Since(resolveStart)
	if !quietMode {
		fmt.Printf("Resolution DNS: %v\n\n", stats.ResolveTime)
		fmt.Println("--- Debut de la sequence de ping ---")
	}

	err = pingWithInterface(host, iface, sourceIP, count, interval, quietMode, stats)
	if err != nil {
		if !quietMode {
			fmt.Printf("Erreur fatale: %v\n", err)
		}
		os.Exit(1)
	}

	calculateStats(stats)

	if !quietMode {
		printDetailedReport(stats)
	}

	if mcStore != nil {
		err := mcStore.SavePingStats(stats, quietMode)
		if err != nil {
			if !quietMode {
				fmt.Printf("\nErreur de sauvegarde Memcached : %v\n", err)
			}
			os.Exit(1)
		}
	}
}

func getInterfaceByIP(ip string) string {
	target := net.ParseIP(ip)
	if target == nil {
		return ""
	}

	interfaces, err := net.Interfaces()
	if err != nil {
		return ""
	}

	for _, iface := range interfaces {
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, addr := range addrs {
			if ipnet, ok := addr.(*net.IPNet); ok {
				if ipnet.IP.Equal(target) {
					return iface.Name
				}
			}
		}
	}
	return ""
}

func getInterfaceIP(name string) (string, error) {
	iface, err := net.InterfaceByName(name)
	if err != nil {
		return "", err
	}

	addrs, err := iface.Addrs()
	if err != nil {
		return "", err
	}

	for _, addr := range addrs {
		if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				return ipnet.IP.String(), nil
			}
		}
	}

	return "", fmt.Errorf("aucune adresse IPv4 trouvee sur l'interface %s", name)
}

func listInterfaces() {
	interfaces, err := net.Interfaces()
	if err != nil {
		fmt.Printf("Erreur lors de la liste des interfaces : %v\n", err)
		return
	}

	for _, iface := range interfaces {
		if iface.Flags&net.FlagUp == 0 {
			continue
		}
		if iface.Flags&net.FlagLoopback != 0 {
			continue
		}

		addrs, _ := iface.Addrs()
		var ips []string
		for _, addr := range addrs {
			if ipnet, ok := addr.(*net.IPNet); ok {
				if ipnet.IP.To4() != nil {
					ips = append(ips, ipnet.IP.String())
				}
			}
		}

		if len(ips) > 0 {
			fmt.Printf("  %-10s %s\n", iface.Name, strings.Join(ips, ", "))
		}
	}
}

func calculateStats(s *PingStats) {
	s.Lost = s.Sent - s.Received
	s.LossPercent = float64(s.Lost) / float64(s.Sent) * 100

	if s.Received == 0 {
		s.MOS = calculateMOS(s)
		return
	}

	sort.Slice(s.RTTs, func(i, j int) bool {
		return s.RTTs[i] < s.RTTs[j]
	})

	s.MinRTT = s.RTTs[0]
	s.MaxRTT = s.RTTs[len(s.RTTs)-1]

	var sum time.Duration
	for _, rtt := range s.RTTs {
		sum += rtt
	}
	s.AvgRTT = sum / time.Duration(len(s.RTTs))

	var varianceSum float64
	avgFloat := float64(s.AvgRTT)
	for _, rtt := range s.RTTs {
		diff := float64(rtt) - avgFloat
		varianceSum += diff * diff
	}
	s.StdDev = time.Duration(sqrtFloat64(varianceSum / float64(len(s.RTTs))))

	if len(s.RTTs) > 1 {
		var jitterSum time.Duration
		for i := 1; i < len(s.RTTs); i++ {
			diff := s.RTTs[i] - s.RTTs[i-1]
			if diff < 0 {
				diff = -diff
			}
			jitterSum += diff
		}
		s.Jitter = jitterSum / time.Duration(len(s.RTTs)-1)
	}

	s.MOS = calculateMOS(s)
}

func sqrtFloat64(x float64) float64 {
	if x == 0 {
		return 0
	}
	z := x
	for i := 0; i < 10; i++ {
		z = (z + x/z) / 2
	}
	return z
}

func repeatChar(char string, n int) string {
	return strings.Repeat(char, n)
}

func printDetailedReport(s *PingStats) {
	fmt.Println("\n" + repeatChar("=", 60))
	fmt.Println("           RAPPORT D'EVALUATION DE LIEN RESEAU")
	fmt.Println(repeatChar("=", 60))

	fmt.Printf("\nCible :         %s\n", s.Host)
	if s.Interface != "" {
		fmt.Printf("Interface :     %s (%s)\n", s.Interface, s.SourceIP)
	} else if s.SourceIP != "" {
		fmt.Printf("IP Source :     %s\n", s.SourceIP)
	}
	fmt.Printf("Paquets Envoyes : %d\n", s.Sent)
	fmt.Printf("Paquets Recus :   %d\n", s.Received)
	fmt.Printf("Paquets Perdus :  %d (%.2f%%)\n", s.Lost, s.LossPercent)

	fmt.Println("\n" + repeatChar("-", 40))
	fmt.Println("EVALUATION DE LA QUALITE DU LIEN")
	fmt.Println(repeatChar("-", 40))

	if s.LossPercent == 0 {
		fmt.Println("Perte de Paquets : EXCELLENT (0%)")
	} else if s.LossPercent < 1 {
		fmt.Println("Perte de Paquets : BON (<1%)")
	} else if s.LossPercent < 5 {
		fmt.Println("Perte de Paquets : MOYEN (1-5%)")
	} else {
		fmt.Println("Perte de Paquets : FAIBLE (>5%)")
	}

	if s.Received > 0 {
		avgMs := float64(s.AvgRTT) / float64(time.Millisecond)
		fmt.Printf("\nMETRIQUES DE LATENCE :\n")
		fmt.Printf("   Minimum :    %10.3f ms\n", float64(s.MinRTT)/float64(time.Millisecond))
		fmt.Printf("   Maximum :    %10.3f ms\n", float64(s.MaxRTT)/float64(time.Millisecond))
		fmt.Printf("   Moyenne :    %10.3f ms", avgMs)

		if avgMs < 10 {
			fmt.Println(" [Excellent]")
		} else if avgMs < 30 {
			fmt.Println(" [Bon]")
		} else if avgMs < 100 {
			fmt.Println(" [Moyen]")
		} else {
			fmt.Println(" [Faible]")
		}

		fmt.Printf("   Ecart-type : %10.3f ms (variabilite)\n",
			float64(s.StdDev)/float64(time.Millisecond))
		fmt.Printf("   Gigue :      %10.3f ms (coherence)\n",
			float64(s.Jitter)/float64(time.Millisecond))

		fmt.Println("\nCOHERENCE :")
		consistency := float64(s.StdDev) / float64(s.AvgRTT) * 100
		fmt.Printf("   Coefficient de Variation : %.2f%%\n", consistency)
		if consistency < 5 {
			fmt.Println("   Etat : TRES STABLE")
		} else if consistency < 15 {
			fmt.Println("   Etat : STABLE")
		} else if consistency < 30 {
			fmt.Println("   Etat : VARIABLE")
		} else {
			fmt.Println("   Etat : INSTABLE")
		}

		if len(s.RTTs) > 0 {
			fmt.Println("\nPERCENTILES :")
			p50 := s.RTTs[len(s.RTTs)*50/100]
			p90 := s.RTTs[len(s.RTTs)*90/100]
			p95 := s.RTTs[len(s.RTTs)*95/100]
			p99 := s.RTTs[len(s.RTTs)*99/100]

			fmt.Printf("   50e (mediane) : %10.3f ms\n", float64(p50)/float64(time.Millisecond))
			fmt.Printf("   90e :           %10.3f ms\n", float64(p90)/float64(time.Millisecond))
			fmt.Printf("   95e :           %10.3f ms\n", float64(p95)/float64(time.Millisecond))
			fmt.Printf("   99e :           %10.3f ms\n", float64(p99)/float64(time.Millisecond))
		}

		if s.Received > 0 {
			mos := s.MOS
			fmt.Printf("\nQUALITE VoIP ESTIMEE (MOS) : %.1f/5.0\n", mos)
			if mos >= 4.3 {
				fmt.Println("   Qualite : Excellente (VoIP/Premium)")
			} else if mos >= 4.0 {
				fmt.Println("   Qualite : Elevee (VoIP/HD)")
			} else if mos >= 3.6 {
				fmt.Println("   Qualite : Moyenne (VoIP/Acceptable)")
			} else if mos >= 3.1 {
				fmt.Println("   Qualite : Faible (VoIP/Marginale)")
			} else {
				fmt.Println("   Qualite : Mauvaise (Inadaptee a la VoIP)")
			}
		}
	}

	fmt.Printf("\nResolution DNS : %v\n", s.ResolveTime)

	if len(s.Errors) > 0 {
		fmt.Println("\nERREURS RENCONTREES :")
		for _, err := range s.Errors {
			fmt.Printf("   - %s\n", err)
		}
	}

	fmt.Println("\n" + repeatChar("=", 60))
}

func calculateMOS(s *PingStats) float64 {
	if s.Received == 0 {
		return 1.0
	}

	latency := float64(s.AvgRTT) / float64(time.Millisecond)
	jitter := float64(s.Jitter) / float64(time.Millisecond)
	packetLoss := s.LossPercent

	R := 93.2 - (latency * 0.024) - (jitter * 0.11) - (packetLoss * 2.5)

	if R < 0 {
		R = 0
	}
	if R > 100 {
		R = 100
	}

	mos := 1 + (0.035 * R) + (0.000007 * R * (R - 60) * (100 - R))
	if mos > 5 {
		return 5.0
	}
	if mos < 1 {
		return 1.0
	}
	return mos
}
