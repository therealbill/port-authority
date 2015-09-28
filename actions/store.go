package actions

import (
	"errors"
	"fmt"
	"log"
	"strconv"

	"github.com/therealbill/libredis/client"
)

var (
	rediscon         *client.Redis
	redisInitialized bool
	eventexpiration  int
)

func InitializeRedisClient(address, auth string) (err error) {
	rediscon, err = client.DialWithConfig(&client.DialConfig{Address: address, Password: auth})
	if err != nil {
		log.Print("Failed InitializeRedisClient with err: ", err.Error())
		return err
	}
	redisInitialized = true
	return nil
}

func RedisConnection() (*client.Redis, error) {
	if !redisInitialized {
		return rediscon, errors.New("Need to call InitializeRedisClient first!")
	}
	return rediscon, nil
}

func InitializePorts(start, end int) error {
	rc, err := RedisConnection()
	if err != nil {
		return err
	}
	exists, err := rc.Exists("open_ports")
	if err != nil {
		return err
	}
	if exists {
		return errors.New("The backend has already been initialized, so I won't re-initialize it")
	}
	for i := start; i < end; i++ {
		rc.SAdd("open_ports", fmt.Sprintf("%d", i))
	}
	added, err := rc.SCard("open_ports")
	needed := end - start
	if added != int64(needed) {
		errm := fmt.Sprintf("Needed %d ports initialized, got %d", needed, added)
		log.Print(errm)
		return fmt.Errorf(errm)
	}
	return nil
}

func GetOpenPort(iname string) (int, error) {
	rc, err := RedisConnection()
	if err != nil {
		return 0, err
	}
	already_there, _ := rc.HGet("i2port", iname)
	aport := string(already_there)
	if len(aport) > 0 {
		log.Printf("An open port request for '%s' was made, but i already have a port for it in the i2port, returning it.", iname)
		iport, _ := strconv.Atoi(aport)
		return iport, nil
	}

	bport, err := rc.SPop("open_ports")
	port := string(bport)
	if err != nil {
		log.Printf("Error on SPOP: %v", err)
		return 0, err
	}

	added, err := rc.SAdd("assigned_ports", string(port))
	if err != nil {
		log.Printf("Error on SAdd '%d' to 'assigned_ports': %v", port, err)
		return 0, err
	}

	if added == 0 {
		log.Printf("Error on SAdd '%d' already in 'assigned_ports'! This likely means something didn't get cleaned up.", port)
		assigned_to, _ := rc.HGet("i2port", port)
		if len(assigned_to) != 0 {
			log.Printf("Assigned_to: %s (%d)", assigned_to, len(assigned_to))
			log.Printf("Looks like this port as previously assigned to '%s'. I am now going to abort to avoid stomping on things.", assigned_to)
			return 0, fmt.Errorf("Port obtained from GetOpenPort returned a port listed as assigned.")
		} else {
			log.Printf("Looks like this port was in the assigned_ports set, but no mapping id was found. This is sucky but non-fatal so I will continue to do my job. Someone does need to look into why this happened.", assigned_to)
		}
	}

	isnew, err := rc.HSetnx("i2port", iname, port)
	if !isnew {
		assigned_to, _ := rc.HGet("i2port", iname)
		log.Printf("Looks like this id was previously assigned the port '%s', though it was not in the `assigned_ports` set. I am now going return the pulled port to the pool and return the previously assigned port.", assigned_to)
		already_there, _ := rc.HGet("i2port", iname)
		oport, _ := strconv.Atoi(string(already_there))
		// If we reach here this should already be done by the other process but we can do it here to ensure it is correct
		rc.SAdd("open_ports", port)
		rc.SAdd("assigned_ports", port)
		return oport, nil
	}
	isnew, err = rc.HSetnx("port2i", port, iname)
	if !isnew {
		assigned_to, _ := rc.HGet("i2port", iname)
		em := fmt.Errorf("Looks like this port was previously assigned the id '%s', though it was not in the `assigned_ports` set. I am now going do a full error and abor tbecause somethis is rotten in Denmark, Bob. Someone need to look into this imediately!", assigned_to)
		log.Print(em.Error())
		return 0, em
	}
	iport, _ := strconv.Atoi(port)

	return iport, nil
}

func GetInstanceFromPort(port int) (iname string, err error) {
	rc, err := RedisConnection()
	if err != nil {
		return "", err
	}
	id, err := rc.HGet("port2i", fmt.Sprintf("%d", port))
	if err != nil {
		return "", err
	}
	return string(id), nil
}

func GetPortFromInstance(id string) (port string, err error) {
	rc, err := RedisConnection()
	if err != nil {
		return "", err
	}
	bport, err := rc.HGet("i2port", id)
	if err != nil {
		return "", err
	}
	port = string(bport)
	return port, nil
}

func GetOpenPortCount() (int64, error) {
	rc, err := RedisConnection()
	if err != nil {
		return 0, err
	}
	return rc.SCard("open_ports")
}

func GetOpenPortList() (ports []string, err error) {
	rc, err := RedisConnection()
	if err != nil {
		return ports, err
	}
	return rc.SMembers("open_ports")
}

func GetReservedPortCount() (int64, error) {
	rc, err := RedisConnection()
	if err != nil {
		return 0, err
	}
	return rc.SCard("assigned_ports")
}

func GetReservedPortList() (ports []string, err error) {
	rc, err := RedisConnection()
	if err != nil {
		return ports, err
	}
	return rc.SMembers("assigned_ports")
}

func RemoveService(id string) error {
	rc, err := RedisConnection()
	if err != nil {
		return err
	}
	port, err := rc.HGet("i2port", id)
	if len(port) == 0 { // it isn't there to be deleted
		log.Print("del:", port)
		return nil
	}
	tc, err := rc.Transaction()
	if err != nil {
		log.Printf("Failed to start Redis transaction. Error: %v", err)
		return err
	}
	defer tc.Close()
	// remove from maps
	tc.Command("HDEL", "i2port", id)
	tc.Command("HDEL", "port2i", id)
	//remove from assigned_ports
	tc.Command("SREM", "assigned_ports", string(port))
	tc.Exec()
	rc.SAdd("open_ports", string(port))
	return nil
}
