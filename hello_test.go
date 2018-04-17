package cryptotrade

import "testing"

func TestHello(t *testing.T) {
    if "Hello aaa" != Hello("aaa"){
        t.Errorf("aaa")
    }
}

